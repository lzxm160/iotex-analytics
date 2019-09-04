
CREATE TABLE IF NOT EXISTS `account_history`  (
	`address` varchar(41) NOT NULL,
    `block_height` decimal(65, 0) NOT NULL,
  	`nonce` decimal(65, 0) NOT NULL,
  	`balance` text,
  	`root` varchar(64),
  	`code_hash` varchar(64),
  	`code` text,
  	`is_candidate` text,
  	`voting_weight` text,
  PRIMARY KEY `address_block_height`(`address`, `block_height`),
  INDEX `address_block_height`(`address`, `block_height`)
)
--contract trie's data how to save