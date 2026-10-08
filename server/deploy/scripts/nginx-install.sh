#!/usr/bin/env bash
#
# Render the nginx templates for your domain and install them. Run ON the
# server after `make sync-deploy`:
#
#   DOMAIN=example.com bash ~/scripts/nginx-install.sh
#
# Produces api.<DOMAIN>, mcp.<DOMAIN>, grafana.<DOMAIN> and rabbitmq.<DOMAIN>
# server blocks. Set HOSTS to install fewer, e.g. HOSTS="api mcp" to keep the
# admin consoles off the internet entirely (reach them over an SSH tunnel).
#
# Re-running overwrites the files, which drops the TLS blocks certbot added —
# run certbot again afterwards (it reuses existing certificates).

set -euo pipefail

: "${DOMAIN:?set DOMAIN, e.g. DOMAIN=example.com}"
HOSTS="${HOSTS:-api mcp grafana rabbitmq}"
SRC="${SRC:-$HOME/deploy/nginx/conf.d}"
DEST="${DEST:-/etc/nginx/conf.d}"

command -v envsubst >/dev/null || { echo "envsubst not found (apt-get install gettext-base)" >&2; exit 1; }

sudo rm -f "$DEST/default.conf"
sudo cp "$SRC/00-shared.conf" "$DEST/00-shared.conf"
for h in $HOSTS; do
  tmpl="$SRC/$h.conf.template"
  [ -f "$tmpl" ] || { echo "no template $tmpl" >&2; exit 1; }
  # Restrict substitution to ${DOMAIN} so nginx's own $variables survive.
  DOMAIN="$DOMAIN" envsubst '${DOMAIN}' < "$tmpl" | sudo tee "$DEST/$h.conf" > /dev/null
  echo "  ok $h.$DOMAIN -> $DEST/$h.conf"
done

sudo nginx -t && sudo systemctl reload nginx

echo
echo "Next, issue certificates (DNS for each host must already point here):"
printf '  sudo certbot --nginx'
for h in $HOSTS; do printf ' -d %s.%s' "$h" "$DOMAIN"; done
echo
