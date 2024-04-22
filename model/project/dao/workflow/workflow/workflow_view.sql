( SELECT *
  FROM WORKFLOW w
  WHERE w.PROJECT_ID = ${criteria.AppendBinding($Unsafe.ProjectID)}
     ${predicate.Builder().CombineOr($predicate.FilterGroup(0, "AND")).Build("AND")} )