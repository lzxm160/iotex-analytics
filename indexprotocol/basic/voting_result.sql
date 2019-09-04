
CREATE TABLE IF NOT EXISTS `voting_result`  (
  `epoch_number` decimal(65, 0) NOT NULL,
  `delegate_name` varchar(255) NOT NULL,
  `operator_address` varchar(41) NOT NULL,
  `reward_address` varchar(41) NOT NULL,
  `total_weighted_votes` decimal(65, 0) NOT NULL,
  `self_staking` decimal(65, 0) NOT NULL,
  `block_reward_percentage` int(11) NULL DEFAULT 100,
  `epoch_reward_percentage` int(11) NULL DEFAULT 100,
  `foundation_bonus_percentage` int(11) NULL DEFAULT 100,
  `staking_address` varchar(40) NULL DEFAULT '0',
  UNIQUE INDEX `epoch_candidate_index`(`epoch_number`, `delegate_name`)
) 