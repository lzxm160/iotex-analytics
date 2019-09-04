
CREATE TABLE IF NOT EXISTS `receipt_history`  (
  `status` decimal(65, 0) NOT NULL,
  `block_height` decimal(65, 0) NOT NULL,
  `action_hash` varchar(64) NOT NULL,
  `receipt_hash` varchar(64) NOT NULL,
  `gas_consumed` decimal(65, 0) NOT NULL,
  `contract_address` varchar(41),
  `logs` text,--serialize all logs
  PRIMARY KEY (`action_hash`),
  INDEX `action_hash`(`action_hash`)
)