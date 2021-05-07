#!/bin/zsh

remote=$1
homedir=$2

if [[ -z $remote ]] || [[ -z $homedir ]]; then
  echo "usage: ./sync_everything.sh <remote> <homedir>"
fi

curr_dir=$(pwd)

cd $HOME/go/src/github.com || exit

cd bruno-anjos || exit
echo "Syncing CED..."
rsync -a cloud-edge-deployment/ "$remote":"$homedir"/go/src/github.com/bruno-anjos/cloud-edge-deployment --delete

echo "Syncing ArchimedesHTTPClient..."
rsync -a archimedesHTTPClient/ "$remote":"$homedir"/go/src/github.com/bruno-anjos/archimedesHTTPClient --delete

cd .. || exit
echo "Syncing NOVAPokemon..."
rsync -a NOVAPokemon/ "$remote":"$homedir"/go/src/github.com/NOVAPokemon --delete --exclude "*.tar" \
  --exclude "*.git/*" --exclude "venv/*"

cd nm-morais || exit
echo "Syncing go-babel..."
rsync -a go-babel/ "$remote":"$homedir"/go/src/github.com/nm-morais/go-babel --delete

echo "Syncing demmon-common..."
rsync -a demmon-common/ "$remote":"$homedir"/go/src/github.com/nm-morais/demmon-common --delete

echo "Syncing demmon-client..."
rsync -a demmon-client/ "$remote":"$homedir"/go/src/github.com/nm-morais/demmon-client --delete

echo "Syncing demmon-exporter..."
rsync -a demmon-exporter/ "$remote":"$homedir"/go/src/github.com/nm-morais/demmon-exporter --delete

echo "Syncing demmon..."
rsync -a demmon/ "$remote":"$homedir"/go/src/github.com/nm-morais/demmon --delete

cd "$curr_dir" || exit
