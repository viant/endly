select * from WORKFLOW
WHERE $criteria.In("ID", $CurWorkflowId.Values)