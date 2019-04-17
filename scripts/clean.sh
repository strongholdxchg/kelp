#!/bin/bash

function delete_large_dir() {
    if [ ! -d "$1" ]
    then
        return
    fi

    rm -rf $1
    echo $1
}

if [[ ($# -gt 1 || `pwd | rev | cut -d'/' -f1 | rev` != "kelp") ]]
then
    echo "need to invoke from the root 'kelp' directory"
    exit 1
fi

echo "removing files ..."
rm -vrf bin
rm -vrf build
delete_large_dir gui/web/build
delete_large_dir gui/web/node_modules
rm -vf gui/filesystem_vfsdata_release.go
echo "... done"
