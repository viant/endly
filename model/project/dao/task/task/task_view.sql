( SELECT *
  FROM TASK t
  WHERE 1=1
  ${predicate.Builder().CombineOr($predicate.FilterGroup(0, "AND")).Build("AND")} )