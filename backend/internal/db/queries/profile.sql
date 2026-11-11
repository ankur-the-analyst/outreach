-- name: InsertProfile :one
insert into profiles (workspace_id, id, name, about)
values ($1, $2, $3, $4)
returning workspace_id, id, name, about, links, created_at;

-- name: ListProfiles :many
select workspace_id, id, name, about, links, created_at
from profiles
where workspace_id = $1
order by created_at desc;

-- name: GetProfile :one
select workspace_id, id, name, about, links, created_at
from profiles
where workspace_id = $1 and id = $2;

-- name: NextResumeVersion :one
select coalesce(max(version), 0)::int + 1
from resumes
where workspace_id = $1 and profile_id = $2;

-- name: InsertResume :one
insert into resumes (
  workspace_id, id, profile_id, version, file_key, file_name, mime, text_content
) values (
  $1, $2, $3, $4, $5, $6, $7, $8
)
returning workspace_id, id, profile_id, version, file_key, file_name, mime, text_content, created_at;

-- name: LatestResume :one
select workspace_id, id, profile_id, version, file_key, file_name, mime, text_content, created_at
from resumes
where workspace_id = $1 and profile_id = $2
order by version desc
limit 1;

-- name: GetResume :one
select workspace_id, id, profile_id, version, file_key, file_name, mime, text_content, created_at
from resumes
where workspace_id = $1 and id = $2;

-- name: InsertJob :one
insert into jobs (workspace_id, id, source_url, raw_text, title)
values ($1, $2, $3, $4, $5)
returning workspace_id, id, source_url, raw_text, title, analysis, analysis_status, created_at;

-- name: GetJob :one
select workspace_id, id, source_url, raw_text, title, analysis, analysis_status, created_at
from jobs
where workspace_id = $1 and id = $2;

-- name: ListJobs :many
select workspace_id, id, source_url, raw_text, title, analysis, analysis_status, created_at
from jobs
where workspace_id = $1
  and (sqlc.narg('after')::uuid is null or id < sqlc.narg('after'))
order by id desc
limit sqlc.arg('n');

-- name: UpdateJobAnalysis :one
update jobs
set analysis = $3, analysis_status = $4
where workspace_id = $1 and id = $2
returning workspace_id, id, source_url, raw_text, title, analysis, analysis_status, created_at;

-- name: UpsertFit :one
insert into fits (
  workspace_id, id, job_id, profile_id, resume_id, score, result
) values (
  $1, $2, $3, $4, $5, $6, $7
)
on conflict (workspace_id, job_id, profile_id) do update
set resume_id = excluded.resume_id, score = excluded.score, result = excluded.result
returning workspace_id, id, job_id, profile_id, resume_id, score, result, created_at;

-- name: ListFits :many
select workspace_id, id, job_id, profile_id, resume_id, score, result, created_at
from fits
where workspace_id = $1
order by created_at desc;

-- name: DeleteFitsForWorkspace :exec
delete from fits where workspace_id = $1;

-- name: DeleteResumesForWorkspace :exec
delete from resumes where workspace_id = $1;

-- name: DeleteJobsForWorkspace :exec
delete from jobs where workspace_id = $1;

-- name: DeleteProfilesForWorkspace :exec
delete from profiles where workspace_id = $1;
