SELECT *
  FROM ASSET
WHERE $criteria.In("ID", $CurWorkflowAssetId.Values)