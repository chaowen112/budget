DELETE FROM asset_types at
WHERE at.name IN ('Mutual Fund', 'Robo-Advisor', '401k', 'IRA')
  AND NOT EXISTS (
    SELECT 1
    FROM assets a
    WHERE a.asset_type_id = at.id
  );
