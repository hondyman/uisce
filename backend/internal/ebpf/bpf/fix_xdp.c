// internal/ebpf/fix_xdp.c
// SPDX-License-Identifier: GPL-2.0
// eBPF XDP program to parse FIX Protocol Tag 35=D (New Order Single)
// at the NIC ring-buffer layer, bypassing the Linux TCP/IP network stack.
// Compiled with: clang -target bpf -O2 -c fix_xdp.c

#include <linux/bpf.h>
#include <linux/if_ether.h>
#include <linux/ip.h>
#include <linux/tcp.h>
#include <linux/pkt_cls.h>
#include <bpf/bpf_helpers.h>
#include <bpf/bpf_endian.h>
#include <stdint.h>

#define FIX_PORT 8980
#define SOH 0x01

struct {
    __uint(type, BPF_MAP_TYPE_RINGBUF);
    __uint(max_entries, 1024 * 1024); /* 1MB Ring Buffer */
} fix_events SEC(".maps");

struct fix_order_event {
    __u64 timestamp_ns;
    char  cl_ord_id[32];
    char  symbol[16];
    __u32 quantity;
    __u32 price_raw; /* Price multiplied by 10000 for fixed-point representation */
    __u8  side;
};

/*
 * find_fix_tag searches for a FIX tag pattern "tag=" within a bounded memory region.
 * Returns pointer to the character after '=', or NULL if not found.
 */
static __always_inline char *find_fix_tag(void *data_start, void *data_end, char tag0, char tag1) {
    char *p = data_start;
    while (p + 2 < data_end) {
        if (p[0] == tag0 && p[1] == tag1 && p[2] == '=') {
            return p + 3;
        }
        p++;
    }
    return NULL;
}

/*
 * extract_fix_uint32 extracts a decimal number from a FIX tag value field.
 * Stops at SOH (0x01) or end of buffer.
 */
static __always_inline __u32 extract_fix_uint32(char *val_start, char *val_end) {
    __u32 result = 0;
    char *p = val_start;
    while (p < val_end && *p != SOH && *p >= '0' && *p <= '9') {
        result = result * 10 + (__u32)(*p - '0');
        p++;
    }
    return result;
}

/*
 * copy_fix_field copies a FIX field value into a fixed-size char buffer.
 * Stops at SOH or buffer limit.
 */
static __always_inline void copy_fix_field(char *dst, int dst_len, char *src_start, char *src_end) {
    int i = 0;
    while (i < dst_len - 1 && src_start < src_end && *src_start != SOH) {
        dst[i++] = *src_start++;
    }
    dst[i] = '\0';
}

SEC("xdp")
int xdp_fix_parser(struct xdp_md *ctx) {
    void *data_end = (void *)(long)ctx->data_end;
    void *data     = (void *)(long)ctx->data;

    /* 1. Parse Ethernet header */
    struct ethhdr *eth = data;
    if ((void *)(eth + 1) > data_end)
        return XDP_PASS;
    if (eth->h_proto != bpf_htons(ETH_P_IP))
        return XDP_PASS;

    /* 2. Parse IP header */
    struct iphdr *ip = (void *)(eth + 1);
    if ((void *)(ip + 1) > data_end)
        return XDP_PASS;
    if (ip->protocol != IPPROTO_TCP)
        return XDP_PASS;

    /* 3. Parse TCP header */
    struct tcphdr *tcp = (void *)((unsigned char *)ip + (ip->ihl * 4));
    if ((void *)(tcp + 1) > data_end)
        return XDP_PASS;
    if (bpf_ntohs(tcp->dest) != FIX_PORT)
        return XDP_PASS;

    /* 4. Locate FIX payload start (after TCP header) */
    unsigned int tcp_hdr_len = tcp->doff * 4;
    char *payload = (char *)tcp + tcp_hdr_len;
    unsigned int payload_len = (unsigned int)(data_end - payload);
    if (payload_len < 16)
        return XDP_PASS;

    /* 5. Verify Tag 35=D (MsgType = New Order Single) */
    char *msg_type_tag = find_fix_tag(payload, data_end, '3', '5');
    if (!msg_type_tag || *msg_type_tag != 'D')
        return XDP_PASS;

    /* 6. Allocate ring buffer event */
    struct fix_order_event *evt;
    evt = bpf_ringbuf_reserve(&fix_events, sizeof(*evt), 0);
    if (!evt)
        return XDP_PASS;

    /* 7. Populate timestamp */
    evt->timestamp_ns = bpf_ktime_get_ns();

    /* 8. Extract ClOrdID (Tag 11) */
    char *cl_ord_tag = find_fix_tag(msg_type_tag + 1, data_end, '1', '1');
    if (cl_ord_tag) {
        char *cl_ord_end = cl_ord_tag;
        while (cl_ord_end < data_end && *cl_ord_end != SOH)
            cl_ord_end++;
        copy_fix_field(evt->cl_ord_id, 32, cl_ord_tag, cl_ord_end);
    } else {
        evt->cl_ord_id[0] = '\0';
    }

    /* 9. Extract Symbol (Tag 55) */
    char *symbol_tag = find_fix_tag(msg_type_tag + 1, data_end, '5', '5');
    if (symbol_tag) {
        char *symbol_end = symbol_tag;
        while (symbol_end < data_end && *symbol_end != SOH)
            symbol_end++;
        copy_fix_field(evt->symbol, 16, symbol_tag, symbol_end);
    } else {
        evt->symbol[0] = '\0';
    }

    /* 10. Extract OrderQty (Tag 38) */
    char *qty_tag = find_fix_tag(msg_type_tag + 1, data_end, '3', '8');
    if (qty_tag) {
        char *qty_end = qty_tag;
        while (qty_end < data_end && *qty_end != SOH)
            qty_end++;
        evt->quantity = extract_fix_uint32(qty_tag, qty_end);
    } else {
        evt->quantity = 0;
    }

    /* 11. Extract Price (Tag 44) - stored as price_raw * 10000 */
    char *price_tag = find_fix_tag(msg_type_tag + 1, data_end, '4', '4');
    if (price_tag) {
        char *price_end = price_tag;
        while (price_end < data_end && *price_end != SOH)
            price_end++;
        /* Parse as integer scaled by 10000 */
        __u64 price_scaled = 0;
        char *p = price_tag;
        while (p < price_end && *p >= '0' && *p <= '9') {
            price_scaled = price_scaled * 10 + (__u64)(*p - '0');
            p++;
        }
        /* Detect decimal point and apply scale */
        if (p < price_end && *p == '.') {
            p++;
            __u64 fraction = 0;
            int frac_digits = 0;
            while (p < price_end && *p >= '0' && *p <= '9' && frac_digits < 4) {
                fraction = fraction * 10 + (__u64)(*p - '0');
                p++;
                frac_digits++;
            }
            while (frac_digits < 4) {
                fraction *= 10;
                frac_digits++;
            }
            price_scaled = price_scaled * 10000 + fraction;
        } else {
            price_scaled *= 10000;
        }
        evt->price_raw = (__u32)price_scaled;
    } else {
        evt->price_raw = 0;
    }

    /* 12. Extract Side (Tag 54) */
    char *side_tag = find_fix_tag(msg_type_tag + 1, data_end, '5', '4');
    if (side_tag && side_tag < data_end && *side_tag != SOH) {
        evt->side = *side_tag;
    } else {
        evt->side = '0';
    }

    /* 13. Submit to ring buffer for user-space consumption */
    bpf_ringbuf_submit(evt, 0);

    return XDP_PASS;
}

char _license[] SEC("license") = "GPL";
