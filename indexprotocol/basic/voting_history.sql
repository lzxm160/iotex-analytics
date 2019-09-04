
CREATE TABLE IF NOT EXISTS `voting_history`  (
  `epoch_number` decimal(65, 0) NOT NULL,
  `candidate_name` varchar(24) NOT NULL,
  `voter_address` varchar(40) NOT NULL,
  `votes` decimal(65, 0) NOT NULL,
  `weighted_votes` decimal(65, 0) NOT NULL,
  `remaining_duration` text  NOT NULL,
  INDEX `epoch_candidate_index`(`epoch_number`, `candidate_name`),
  INDEX `epoch_voter_index`(`epoch_number`, `voter_address`),
  INDEX `candidate_voter_index`(`candidate_name`, `voter_address`)
)