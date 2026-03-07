INSERT INTO asset_types (name, category, is_system)
SELECT 'Mutual Fund', 'investment', true
WHERE NOT EXISTS (SELECT 1 FROM asset_types WHERE name = 'Mutual Fund');

INSERT INTO asset_types (name, category, is_system)
SELECT 'Robo-Advisor', 'investment', true
WHERE NOT EXISTS (SELECT 1 FROM asset_types WHERE name = 'Robo-Advisor');

INSERT INTO asset_types (name, category, is_system)
SELECT '401k', 'retirement', true
WHERE NOT EXISTS (SELECT 1 FROM asset_types WHERE name = '401k');

INSERT INTO asset_types (name, category, is_system)
SELECT 'IRA', 'retirement', true
WHERE NOT EXISTS (SELECT 1 FROM asset_types WHERE name = 'IRA');
