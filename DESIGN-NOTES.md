# Datastructures

 - config 
   - user provided configuration, is potentially updated either on startup or by set config rpc
   - configures
     - repos - a list of restic repos to which data may be backed up
     - plans - a list of backup plans which consist of 
       - directories
       - schedule
       - retention policy
  - cache
    - the cache is a local cache of the restic repo's properties e.g. output from listing snapshots, etc. This may be held in ram or on disk? TBD: decide.
  - state
    - state is tracked plan-by-plan and is persisted to disk
      - stores recent operations done for a plan e.g. last backup, last prune, last check, etc.
      - stores status and errors for each plan
      - history is fixed size and is flushed to disk periodically (e.g. every 60 seconds).
    - the state of a repo is the merge of the states of the plans that reference it.