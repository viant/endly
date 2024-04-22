SELECT *
  FROM TASK
WHERE $criteria.In("ID", $CurWorkflowTaskId.Values)