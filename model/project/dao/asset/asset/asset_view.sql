( SELECT *
  FROM ASSET a
  WHERE 1=1
  ${predicate.Builder().CombineOr($predicate.FilterGroup(0, "AND")).Build("AND")} )