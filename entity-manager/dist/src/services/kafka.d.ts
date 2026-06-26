export declare function initKafka(): Promise<void>;
export declare function publishEvent(topic: string, message: any): Promise<void>;
export declare function getKafkaConsumer(): import("kafkajs").Consumer;
export declare function isKafkaReady(): boolean;
export declare function disconnectKafka(): Promise<void>;
//# sourceMappingURL=kafka.d.ts.map