# The Final Pipeline
The final pipeline always runs after the primary pipeline, regardless of whether it failed, and every task in the pipeline runs, regardless of failures. Unlike the primary and fail pipelines, the tasks in the final pipeline run in **FILO** order - first in, last out. This creates a kind of "bracketing" behavior; the "ssh-init" task, for instance, adds a final task to kill the `ssh-agent` at the end of the pipeline. If the "ssh-init" task runs first, the "kill" task will run last; if a following task performs some kind of initialization / setup on a remote host, and adds a final task to clean up, this insures that the remote cleanup occurs before the ssh-agent is killed.

Note that there are several [environment variables](../Environment-Variables.md) that are set at the end of the primary pipeline that can be examined and used for reporting in the final pipeline. For an example of this, see the [finishbuild](https://github.com/lnxjedi/gopherbot/blob/master/tasks/finishbuild.sh) task that runs in the final pipeline of a **GopherCI** build.