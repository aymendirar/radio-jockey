#!/usr/bin/env bash
set -euo pipefail

VPS_HOST="root@radiojockey.live"
VPS_PORT="22"
DEPLOY_PATH="/root/radio-jockey"

REPO_URL="$(git config --get remote.origin.url)"
BRANCH="$(git rev-parse --abbrev-ref HEAD)"

# local:remote pairs — server ships its prod-specific env file, others copy as-is
ENV_FILES=(
  "server/.env.production:server/.env"
  "web/.env:web/.env"
  "discord-jockey/.env:discord-jockey/.env"
  "icecast/.env:icecast/.env"
)

ssh_cmd() {
  ssh -p "$VPS_PORT" "$VPS_HOST" "$@"
}

echo "==> cloning/updating $REPO_URL ($BRANCH) at $VPS_HOST:$DEPLOY_PATH"
ssh_cmd zsh -ls <<EOF
[ -f "\$HOME/.zshrc" ] && source "\$HOME/.zshrc"
set -euo pipefail
mkdir -p ~/.ssh
chmod 700 ~/.ssh
ssh-keyscan -H github.com >> ~/.ssh/known_hosts 2>/dev/null
if [ -d "$DEPLOY_PATH/.git" ]; then
  cd "$DEPLOY_PATH" && git fetch origin && git checkout "$BRANCH" && git pull origin "$BRANCH"
else
  git clone --branch "$BRANCH" "$REPO_URL" "$DEPLOY_PATH"
fi
EOF

echo "==> copying local .env files"
for pair in "${ENV_FILES[@]}"; do
  src="${pair%%:*}"
  dest="${pair#*:}"
  if [ -f "$src" ]; then
    ssh_cmd mkdir -p "$DEPLOY_PATH/$(dirname "$dest")"
    scp -P "$VPS_PORT" "$src" "$VPS_HOST:$DEPLOY_PATH/$dest"
  else
    echo "    skipping $src (not found locally)"
  fi
done

echo "==> running just prod"
ssh_cmd zsh -ls <<EOF
[ -f "\$HOME/.zshrc" ] && source "\$HOME/.zshrc"
set -euo pipefail
cd "$DEPLOY_PATH"
just prod
EOF

echo "==> deploy complete"
