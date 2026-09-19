-- Rollback migration 011
DELETE FROM vend.swift_field_map
WHERE msg_type = 'pacs.008'
  AND field_tag IN (
    'Document/FIToFICstmrCdtTrf/CdtTrfTxInf/PmtId/UETR',
    'Document/FIToFICstmrCdtTrf/CdtTrfTxInf/IntrBkSttlmDt'
  );

UPDATE vend.swift_field_map SET field_tag = 'CdtTrfTxInf/EndToEndId'
WHERE msg_type = 'pacs.008' AND field_tag = 'Document/FIToFICstmrCdtTrf/CdtTrfTxInf/PmtId/EndToEndId';

UPDATE vend.swift_field_map SET field_tag = 'CdtTrfTxInf/Amt/InstdAmt'
WHERE msg_type = 'pacs.008' AND field_tag = 'Document/FIToFICstmrCdtTrf/CdtTrfTxInf/IntrBkSttlmAmt';

UPDATE vend.swift_field_map SET field_tag = 'CdtTrfTxInf/CdtrAgt/FinInstnId/BICFI'
WHERE msg_type = 'pacs.008' AND field_tag = 'Document/FIToFICstmrCdtTrf/CdtTrfTxInf/Cdtr/FinInstnId/BICFI';

UPDATE vend.swift_field_map SET field_tag = 'GrpHdr/MsgId'
WHERE msg_type = 'pacs.008' AND field_tag = 'Document/FIToFICstmrCdtTrf/GrpHdr/MsgId';

UPDATE vend.swift_field_map SET field_tag = 'GrpHdr/CreDtTm'
WHERE msg_type = 'pacs.008' AND field_tag = 'Document/FIToFICstmrCdtTrf/GrpHdr/CreDtTm';
