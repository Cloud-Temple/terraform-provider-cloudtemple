#!/usr/bin/env bash
#
# ct-soak — bounded, breaker-guarded soak probe of ONE read-only Cloud Temple API
# endpoint, to characterize intermittent platform flakiness (e.g. #315: OpenIaaS
# machine_managers 5xx bursts) WITHOUT hammering a shared recette/dev API.
#
# It authenticates with a PAT (client/secret), then fires a fixed quota of GET
# calls per concurrency STEP (1, 2, 4, 8, ...), measuring HTTP code + latency for
# each call. After every step it reports OK% / 5xx / 4xx / errors and latency
# percentiles, and STOPS EARLY if a step's 5xx rate crosses a threshold (the
# bounded-probe / stop-at-first-distress doctrine — never stress to breakage).
#
# READ-ONLY: it only issues GETs (default: the machine_managers list). No writes,
# no orphans. Credentials and the bearer token are NEVER printed.
#
# Usage:
#   scripts/ct-soak.sh                         # default: openiaas machine_managers, steps "1 2 4 8", 25/step (=100)
#   CT_ENV=/path/.env.recette-openiaas scripts/ct-soak.sh
#   STEPS="1 2 4 8 16" PER_STEP=20 scripts/ct-soak.sh
#   API_PATH=/api/compute/v1/open_iaas/storage_repositories scripts/ct-soak.sh   # probe another GET
#
# Env knobs (all optional):
#   CT_ENV          creds file, KEY=value with CLOUDTEMPLE_CLIENT_ID / CLOUDTEMPLE_SECRET_ID
#                   (default: $CT_ENV_OPENIAAS, else ./.env.recette-openiaas)
#   HOST            API host                (default: shiva.cloud-temple.com)
#   API_PATH        the GET path to probe   (default: /api/compute/v1/open_iaas  → machine_managers.list)
#   STEPS           concurrency ladder      (default: "1 2 4 8")
#   PER_STEP        calls per step          (default: 25  → total = PER_STEP * #steps)
#   MAX_TIME        per-call curl timeout s (default: 30)
#   ABORT_5XX_RATE  stop if a step's 5xx rate exceeds this (default: 0.5)
#
# It does NOT mutate anything and is safe to re-run.

set -uo pipefail

HOST="${HOST:-shiva.cloud-temple.com}"
API_PATH="${API_PATH:-/api/compute/v1/open_iaas}"   # OpenIaaS machine_managers list
STEPS="${STEPS:-1 2 4 8}"
PER_STEP="${PER_STEP:-25}"
MAX_TIME="${MAX_TIME:-30}"
ABORT_5XX_RATE="${ABORT_5XX_RATE:-0.5}"

# --- resolve & load credentials (never printed) ------------------------------
CT_ENV="${CT_ENV:-${CT_ENV_OPENIAAS:-./.env.recette-openiaas}}"
if [ ! -f "$CT_ENV" ]; then
  echo "ct-soak: credentials file not found: $CT_ENV" >&2
  echo "         set CT_ENV=/path/to/.env (KEY=value: CLOUDTEMPLE_CLIENT_ID / CLOUDTEMPLE_SECRET_ID)" >&2
  exit 2
fi
set -a; . "$CT_ENV" 2>/dev/null; set +a
if [ -z "${CLOUDTEMPLE_CLIENT_ID:-}" ] || [ -z "${CLOUDTEMPLE_SECRET_ID:-}" ]; then
  echo "ct-soak: CLOUDTEMPLE_CLIENT_ID / CLOUDTEMPLE_SECRET_ID not set in $CT_ENV" >&2
  exit 2
fi

# --- authenticate (PAT → bearer JWT; the response body IS the raw JWT) --------
# The token is captured into a variable and never echoed.
TOKEN="$(curl -ksS --max-time "$MAX_TIME" -X POST \
  "https://${HOST}/api/iam/v2/auth/personal_access_token" \
  -H 'Content-Type: application/json' \
  -d "{\"id\":\"${CLOUDTEMPLE_CLIENT_ID}\",\"secret\":\"${CLOUDTEMPLE_SECRET_ID}\"}" 2>/dev/null)"
case "$TOKEN" in
  eyJ*) : ;;  # looks like a JWT
  *) echo "ct-soak: authentication failed (no JWT returned). Check creds/host." >&2; exit 2 ;;
esac

URL="https://${HOST}${API_PATH}"
echo "ct-soak: target  GET ${URL}"
echo "ct-soak: ladder  STEPS='${STEPS}'  PER_STEP=${PER_STEP}  (total $(( $(wc -w <<<"$STEPS") * PER_STEP )) calls)  abort-5xx-rate=${ABORT_5XX_RATE}"
echo "ct-soak: NOTE read-only probe; stops early if a step's 5xx rate exceeds the threshold."
echo

WORK="$(mktemp -d)"
trap 'rm -rf "$WORK"' EXIT

# one_call <outfile>: GET the endpoint, write "<http_code> <time_total_s>" to outfile.
# A transport error / timeout (curl non-zero) is recorded as code 000.
one_call() {
  local out="$1" line
  line="$(curl -ksS --max-time "$MAX_TIME" -o /dev/null \
            -w '%{http_code} %{time_total}' \
            -H "Authorization: Bearer ${TOKEN}" "$URL" 2>/dev/null)" || line="000 0"
  printf '%s\n' "$line" > "$out"
}

# percentile <p> < sorted_numbers : nearest-rank percentile of stdin (ms, integers)
percentile() {
  local p="$1"
  awk -v p="$p" '{a[NR]=$1} END{ if(NR==0){print "-";exit} n=int((p/100)*NR+0.999); if(n<1)n=1; if(n>NR)n=NR; print a[n] }'
}

TOTAL_CALLS=0; TOTAL_OK=0; TOTAL_5XX=0; TOTAL_4XX=0; TOTAL_ERR=0
aborted=0

printf '%-6s %-6s %-5s %-5s %-5s %-6s %8s %8s %8s %8s\n' \
  STEP CALLS OK 5xx 4xx other "p50(ms)" "p95(ms)" "max(ms)" "OK%"
printf '%.0s-' {1..78}; echo

for C in $STEPS; do
  rm -f "$WORK"/r_*
  done_n=0
  # fire PER_STEP calls in waves of C concurrent requests
  while [ "$done_n" -lt "$PER_STEP" ]; do
    pids=()
    for ((k=0; k<C && done_n<PER_STEP; k++)); do
      one_call "$WORK/r_$done_n" &
      pids+=($!)
      done_n=$((done_n+1))
    done
    wait "${pids[@]}" 2>/dev/null
  done

  # aggregate this step
  step_ok=0; step_5xx=0; step_4xx=0; step_err=0; step_n=0
  : > "$WORK/lat_ms"
  for f in "$WORK"/r_*; do
    [ -f "$f" ] || continue
    read -r code t < "$f"
    step_n=$((step_n+1))
    # latency seconds -> ms (integer)
    ms="$(awk -v t="$t" 'BEGIN{printf "%d", t*1000}')"
    echo "$ms" >> "$WORK/lat_ms"
    case "$code" in
      2??) step_ok=$((step_ok+1)) ;;
      5??) step_5xx=$((step_5xx+1)) ;;
      4??) step_4xx=$((step_4xx+1)) ;;
      *)   step_err=$((step_err+1)) ;;
    esac
  done

  sort -n "$WORK/lat_ms" > "$WORK/lat_sorted"
  p50="$(percentile 50 < "$WORK/lat_sorted")"
  p95="$(percentile 95 < "$WORK/lat_sorted")"
  pmax="$(tail -1 "$WORK/lat_sorted" 2>/dev/null || echo -)"
  okpct="$(awk -v ok="$step_ok" -v n="$step_n" 'BEGIN{ if(n==0){print "-"} else printf "%.0f", 100*ok/n }')"

  printf '%-6s %-6s %-5s %-5s %-5s %-6s %8s %8s %8s %7s%%\n' \
    "$C" "$step_n" "$step_ok" "$step_5xx" "$step_4xx" "$step_err" "$p50" "$p95" "$pmax" "$okpct"

  TOTAL_CALLS=$((TOTAL_CALLS+step_n)); TOTAL_OK=$((TOTAL_OK+step_ok))
  TOTAL_5XX=$((TOTAL_5XX+step_5xx)); TOTAL_4XX=$((TOTAL_4XX+step_4xx)); TOTAL_ERR=$((TOTAL_ERR+step_err))

  # breaker: stop the ladder if this step's 5xx rate crosses the threshold
  trip="$(awk -v x="$step_5xx" -v n="$step_n" -v thr="$ABORT_5XX_RATE" \
    'BEGIN{ if(n>0 && (x/n)>thr) print 1; else print 0 }')"
  if [ "$trip" = "1" ]; then
    echo
    echo "ct-soak: ABORT — step concurrency=$C had 5xx rate > ${ABORT_5XX_RATE} ($step_5xx/$step_n). Stopping the ladder (API distress)."
    aborted=1
    break
  fi
done

echo
printf '%.0s=' {1..78}; echo
echo "TOTAL: calls=$TOTAL_CALLS  ok=$TOTAL_OK  5xx=$TOTAL_5XX  4xx=$TOTAL_4XX  other=$TOTAL_ERR"
awk -v ok="$TOTAL_OK" -v x="$TOTAL_5XX" -v n="$TOTAL_CALLS" 'BEGIN{
  if(n==0){print "TOTAL: no calls"; exit}
  printf "TOTAL: OK rate=%.1f%%  5xx rate=%.1f%%\n", 100*ok/n, 100*x/n
}'
if [ "$TOTAL_5XX" -gt 0 ]; then
  echo "VERDICT: intermittent 5xx observed — consistent with the #315 ComputeManager flakiness. A bounded client retry/backoff on the read path would absorb it."
else
  echo "VERDICT: no 5xx in this run."
fi
[ "$aborted" -eq 1 ] && exit 1 || exit 0
