
CREATE TABLE IF NOT EXISTS `action_history`  (
  `action_type` text NOT NULL,
  `action` text NOT NULL,--serialize action,Transfer,execution etc.
  `action_hash` varchar(64) NOT NULL,
  `receipt_hash` varchar(64) NOT NULL,
  `block_height` decimal(65, 0) NOT NULL,
  `from` varchar(41) NOT NULL,
  `to` varchar(41),
  `gas_price` decimal(65, 0) NOT NULL,
  `gas_consumed` decimal(65, 0) NOT NULL,
  `version` decimal(65, 0) NOT NULL,
  `nonce` decimal(65, 0) NOT NULL,
  `receipt_status` text,
  `senderPubKey` text,
  `signature` text,
  PRIMARY KEY (`action_hash`),
  INDEX `action_hash`(`action_hash`)
)