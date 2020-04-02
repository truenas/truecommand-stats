#!/bin/bash
#Quick script to build the tool for each OS type
mkdir -p binaries/freebsd
mkdir -p binaries/linux
srcname="trueview-stats"
binname="trueview-stats"
cd src-go
for os in freebsd linux
do
  export GOOS="${os}"
  export GOARCH="amd64" #64-bit plaforms only
  #go get
  go build ${srcname}.go
  if [ $? -eq 0 ] ; then
    echo "Build successful: ${os}"
    mv ${srcname} ../binaries/${os}/${binname} 
  else
    echo "Build failed: ${os}"
  fi
done
unset GOOS
unset GOARCH
