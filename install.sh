#!/bin/bash

curl "https://api.github.com/repos/renardev/SpoofDPI-Turkiye/releases/latest" |
    grep '"tag_name":' |
    sed -E 's/.*"([^"]+)".*/\1/' |
    xargs -I {} curl -OL "https://github.com/renardev/SpoofDPI-Turkiye/releases/download/"\{\}"/spoofdpi-${1}.tar.gz"

mkdir -p ~/.spoofdpi/bin

tar -xzvf ./spoofdpi-${1}.tar.gz && \
    rm -rf ./spoofdpi-${1}.tar.gz && \
    mv ./spoofdpi ~/.spoofdpi/bin

if [ $? -ne 0 ]; then
    echo "Error. exiting now"
    exit
fi

export PATH=$PATH:~/.spoofdpi/bin

echo ""
echo "SpoofDPI-Turkiye başarıyla indirildi."
echo "Lütfen rcfile(.bashrc or .zshrc etc..) dosyanızın altına şunu ekleyin:"
echo ""
echo ">>    export PATH=\$PATH:~/.spoofdpi/bin"
