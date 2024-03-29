SELECT *
  FROM ASSET a
  WHERE a.WORKFLOW_ID = '${criteria.AppendBinding($Unsafe.WorkflowID)}'
  ${predicate.Builder().CombineOr($predicate.FilterGroup(0, "AND")).Build("AND")}