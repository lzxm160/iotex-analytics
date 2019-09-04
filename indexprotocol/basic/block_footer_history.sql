
CREATE TABLE IF NOT EXISTS `block_footer_history`  (
  `block_height` decimal(65, 0) NOT NULL,
  `endorsements` text,----serialize all endorsements
  `commitTime` decimal(65, 0) NOT NULL,
  PRIMARY KEY (`block_height`),
  INDEX `block_height`(`block_height`)
)