#!/bin/zsh

curr_dir=$(pwd)

cd $HOME/go/src/github.com || exit

cd bruno-anjos || exit
echo "Syncing CED..."
rsync -a cloud-edge-deployment/ dicluster:/home/b.anjos/go/src/github.com/bruno-anjos/cloud-edge-deployment --delete

echo "Syncing Archimedes..."
rsync -a archimedesHTTPClient/ dicluster:/home/b.anjos/go/src/github.com/bruno-anjos/archimedesHTTPClient --delete

cd .. || exit
echo "Syncing NOVAPokemon..."
rsync -a NOVAPokemon/ dicluster:/home/b.anjos/go/src/github.com/NOVAPokemon --delete --exclude "*.tar" --exclude "*.git/*"

cd nm-morais || exit
echo "Syncing go-babel..."
rsync -a go-babel/ dicluster:/home/b.anjos/go/src/github.com/nm-morais/go-babel --delete

echo "Syncing demmon-common..."
rsync -a demmon-common/ dicluster:/home/b.anjos/go/src/github.com/nm-morais/demmon-common --delete

echo "Syncing demmon-client..."
rsync -a demmon-client/ dicluster:/home/b.anjos/go/src/github.com/nm-morais/demmon-client --delete

echo "Syncing demmon-exporter..."
rsync -a demmon-exporter/ dicluster:/home/b.anjos/go/src/github.com/nm-morais/demmon-exporter --delete

echo "Syncing demmon..."
rsync -a demmon/ dicluster:/home/b.anjos/go/src/github.com/nm-morais/demmon --delete

cd "$curr_dir" || exit
