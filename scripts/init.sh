#!/bin/bash

# (return 0 2> /dev/null) && sourced=1 || sourced=0

# if [ "$sourced" -eq 1 ]; then
#   echo "Script is sourced."
# else
#   echo "Script is executed."
# fi
#

# general workflow steps: described in previous step
# make sure you also look for FZF_DEFAULT_OPTS_FILE and load those  before overriding them
# (precedence should go: app defaults -> fzfrc -> config defaults -> module config)
# the default module will be files if one is not specified
# (create a --module command line switch as well as a FZF_MODULE env variable to specify this)

fzfg --init=start
fzfg --init=validate
fzfg --init=config
fzfg --init=rsc=load
fzfg --init=env-load
fzfg --init=env-set

# output debug information:
# summary: summary of init steps
# diffs: formatted pretty config values that differ from defaults
# timings: timing of init steps and overall duration
# envs: important environment variables (FZF_DEFAULT_COMMAND, FZF_DEFAULT_OPTS)
# trace: trace of each config change from snapshots in between steps
# (don't show this unless I specifically ask for it)
fzfg --debug=summary,diffs,timings,envs
