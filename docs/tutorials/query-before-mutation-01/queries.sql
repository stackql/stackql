-- Tutorial 2: Query Before Mutation
-- Validated end to end against Google Cloud Storage.
--
-- Pattern:
-- query live state
-- identify delta
-- policy gate
-- mutate
-- verify convergence


-- ============================================================
-- SETUP
-- ============================================================

REGISTRY PULL google v26.07.00432;


-- ============================================================
-- STEP 1: QUERY LIVE STATE
-- ============================================================

-- Replace your-project-id with the Google Cloud project used for the demo.

SELECT
  name,
  location,
  encryption
FROM google.storage.buckets
WHERE project = 'your-project-id';


-- Buckets where encryption is NULL use Google-managed encryption.
-- For this tutorial, the desired policy is CMEK.

SELECT
  name,
  location,
  encryption
FROM google.storage.buckets
WHERE project = 'your-project-id'
AND encryption IS NULL;


-- ============================================================
-- STEP 2: POLICY GATE
-- ============================================================

-- Gate 1:
-- Only operate on buckets in the expected location.

SELECT
  name,
  location,
  encryption
FROM google.storage.buckets
WHERE project = 'your-project-id'
AND location = 'US'
AND encryption IS NULL;


-- Gate 2:
-- Count the number of resources that would be mutated.
--
-- The automation should stop if this number is higher than
-- the configured safety threshold.

SELECT COUNT(*)
FROM google.storage.buckets
WHERE project = 'your-project-id'
AND location = 'US'
AND encryption IS NULL;


-- ============================================================
-- STEP 3: MUTATE TO CONVERGE
-- ============================================================

-- Apply a customer-managed Cloud KMS key to one non-compliant bucket.
--
-- Replace:
--   demo-app-bucket1
--   your-project-id
--   your-ring
--   your-key
--
-- with values from your environment.

UPDATE google.storage.buckets
SET data__encryption =
  '{"defaultKmsKeyName":"projects/your-project-id/locations/us/keyRings/your-ring/cryptoKeys/your-key"}'
WHERE bucket = 'demo-app-bucket1';


-- Expected result:
-- The operation was despatched successfully


-- ============================================================
-- STEP 4: VERIFY CONVERGENCE
-- ============================================================

SELECT
  name,
  encryption
FROM google.storage.buckets
WHERE bucket = 'demo-app-bucket1';


-- Expected state:
--
-- encryption should now contain a defaultKmsKeyName similar to:
--
-- {
--   "defaultKmsKeyName":
--   "projects/your-project-id/locations/us/keyRings/your-ring/cryptoKeys/your-key"
-- }


-- ============================================================
-- STEP 5: IDEMPOTENCE CHECK
-- ============================================================

SELECT
  name,
  location,
  encryption
FROM google.storage.buckets
WHERE project = 'your-project-id'
AND location = 'US'
AND encryption IS NULL;


-- Once the target bucket has CMEK configured, it should no longer
-- appear in this result set.
--
-- If no non-compliant target remains, no UPDATE should run.
-- Every execution starts from live state and mutates only when
-- the current state differs from policy.
