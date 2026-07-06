#!/usr/bin/env bash
set -euo pipefail

VPS_HOST="root@radiojockey.live"
VPS_PORT="22"
DEPLOY_PATH="/root/radio-jockey"

REPO_URL="$(git config --get remote.origin.url)"
BRANCH="$(git rev-parse --abbrev-ref HEAD)"

ENV_FILES=(
  "server/.env"
  "web/.env"
  "discord-jockey/.env"
  "icecast/.env"
)

ssh_cmd() {
  ssh -p "$VPS_PORT" "$VPS_HOST" "$@"
}

echo "==> cloning/updating $REPO_URL ($BRANCH) at $VPS_HOST:$DEPLOY_PATH"
ssh_cmd bash -s <<EOF
set -euo pipefail
if [ -d "$DEPLOY_PATH/.git" ]; then
  cd "$DEPLOY_PATH" && git fetch origin && git checkout "$BRANCH" && git pull origin "$BRANCH"
else
  git clone --branch "$BRANCH" "$REPO_URL" "$DEPLOY_PATH"
fi
EOF

echo "==> copying local .env files"
for f in "${ENV_FILES[@]}"; do
  if [ -f "$f" ]; then
    ssh_cmd mkdir -p "$DEPLOY_PATH/$(dirname "$f")"
    scp -P "$VPS_PORT" "$f" "$VPS_HOST:$DEPLOY_PATH/$f"
  else
    echo "    skipping $f (not found locally)"
  fi
done

echo "==> installing just (if missing) and running just prod"
ssh_cmd bash -s <<EOF
set -euo pipefail
if ! command -v just >/dev/null 2>&1; then
  curl --proto '=https' --tls-v1.2 -sSf https://just.systems/install.sh | bash -s -- --to "\$HOME/.local/bin"
  export PATH="\$HOME/.local/bin:\$PATH"
fi
cd "$DEPLOY_PATH"
just prod
EOF

echo "==> deploy complete"
