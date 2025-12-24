#!/usr/bin/env bash
set -uo pipefail

ROOT=$(pwd)
TMPHOME="$ROOT/.tmp_home"
mkdir -p "$TMPHOME"
export HOME="$TMPHOME"
CONFIG="$HOME/.gatorconfig.json"
DBURL=${PG_DSN:-"postgres://postgres:postgres@localhost:5432/gator?sslmode=disable"}
cat > "$CONFIG" <<EOF
{"db_url":"$DBURL","current_user_name":""}
EOF

echo "Using DB: $DBURL"

# build binary
if [ ! -f ./gator ]; then
  echo "building gator..."
  go build -o gator .
fi

# register and login (ignore failures so script continues)
./gator register testuser || true
./gator login testuser || true

# add feeds (ignore failures)
./gator addfeed "TechCrunch" "https://techcrunch.com/feed/" || true
./gator addfeed "HN" "https://news.ycombinator.com/rss" || true
./gator addfeed "Boot" "https://blog.boot.dev/index.xml" || true

# run agg in background and capture output
LOGFILE="agg.log"
./gator agg 10s > "$LOGFILE" 2>&1 &
AGG_PID=$!

echo "agg pid=$AGG_PID; sleeping 60s"
sleep 60

echo "killing agg"
kill "$AGG_PID" || true
wait "$AGG_PID" 2>/dev/null || true

echo "agg exited. last 200 lines of log:"
tail -n 200 "$LOGFILE"
