SELECT *
  FROM PROJECT
WHERE $criteria.In("ID", $CurWorkflowProjectId.Values)