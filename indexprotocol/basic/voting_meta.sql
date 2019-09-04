
CREATE TABLE  IF NOT EXISTS `voting_meta`  (
  `epoch_number` decimal(65, 0) NOT NULL,
  `voted_token` decimal(65, 0) NOT NULL,
  `delegate_count` decimal(65, 0) NOT NULL,
  `total_weighted` decimal(65, 0) NOT NULL,
  UNIQUE INDEX `epoch_index`(`epoch_number`)
)