-- agent-orchestrator: managed maki activity package (do not edit)
--
-- Reports Maki turn and status changes to AO through `ao hooks maki <event>`.
-- Inert outside AO sessions: AO sets AO_SESSION_ID for the processes it runs.
local ao_session = maki.uv.os_getenv("AO_SESSION_ID")
if not ao_session or ao_session == "" then
  return
end

local HOOK = 'printf "%s" "$1" | ao hooks maki "$2"'

-- Reporting is best-effort: a missing `ao` or a failed job must never
-- interrupt the Maki session. wait_ms is for teardown, where unawaited jobs
-- are reaped with the session.
local function report(event, session_id, payload, wait_ms)
  payload = payload or {}
  payload.session_id = session_id
  local ok, json = pcall(maki.json.encode, payload)
  if not ok then
    return
  end
  local opts = { stdout = false, stderr = false }
  if not wait_ms and session_id then
    opts.scope = { session = session_id }
  end
  local started, id = pcall(maki.fn.jobstart, { "sh", "-c", HOOK, "sh", json, event }, opts)
  if started and id and wait_ms then
    pcall(maki.fn.jobwait, id, wait_ms)
  end
end

maki.api.create_autocmd("TurnStart", {
  callback = function(ev)
    report("user-prompt-submit", ev.data.session_id)
  end,
})

maki.api.create_autocmd({ "TurnEnd", "TurnError" }, {
  callback = function(ev)
    report("stop", ev.data.session_id, { reason = ev.data.reason })
  end,
})

-- needs_input covers permission prompts and questions to the user.
local waiting = {}
maki.api.create_autocmd("SessionStatusChanged", {
  callback = function(ev)
    local d = ev.data
    if d.focused == false then
      return
    end
    if d.status == "needs_input" and not waiting[d.session_id] then
      waiting[d.session_id] = true
      report("permission-request", d.session_id)
    elseif d.status ~= "needs_input" and waiting[d.session_id] then
      waiting[d.session_id] = nil
      if d.status == "working" then
        report("permission-resolved", d.session_id)
      end
    end
  end,
})

maki.api.create_autocmd("SessionEnd", {
  callback = function(ev)
    if ev.data.reason == "shutdown" then
      report("session-end", ev.data.session_id, nil, math.min(ev.data.deadline_ms or 500, 1000))
    end
  end,
})
