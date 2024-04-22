( SELECT *
  FROM PROJECT p
  WHERE 1=1
  ${predicate.Builder().CombineOr($predicate.FilterGroup(0, "AND")).Build("AND")} )