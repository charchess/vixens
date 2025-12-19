#!/bin/bash
id=`gh pr create -B test -H dev -t "dev to test" -b "" | sed 's/^.*pull\/\(.*\)$/\1/'`
gh pr merge $id -m --auto

sleep 30

id=`gh pr create -B staging -H test -t "test to staging" -b "" | sed 's/^.*pull\/\(.*\)$/\1/'`
gh pr merge $id -m --auto

sleep 30

id=`gh pr create -B main -H staging -t "staging to main" -b "" | sed 's/^.*pull\/\(.*\)$/\1/'`
gh pr merge $id -m --auto

