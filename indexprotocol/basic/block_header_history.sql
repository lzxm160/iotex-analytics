
CREATE TABLE IF NOT EXISTS `block_header_history`  (
  `epoch_number` decimal(65, 0) NOT NULL,
  `block_height` decimal(65, 0) NOT NULL,
  `version` decimal(65, 0) NOT NULL,
  `block_hash` varchar(64) NOT NULL,
  `prevBlockHash` varchar(64) NOT NULL,
  `txRoot` varchar(64) NOT NULL,
  `deltaStateDigest` varchar(64) NOT NULL,
  `receiptRoot` varchar(64) NOT NULL,
  `blockSig` varchar(130) NOT NULL,
  `pubkey` varchar(130) NOT NULL,
  `timestamp` decimal(65, 0) NOT NULL,
  PRIMARY KEY (`block_height`),
  INDEX `block_height`(`block_height`)
)