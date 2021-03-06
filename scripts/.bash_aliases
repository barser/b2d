# Alias definitions.
#
alias df='df -h'
alias du='du -h'

alias ls='ls -p'
alias ll='ls -l'
alias la='ls -la'

alias d='dmenu_run &'
alias ce='cd /etc/sysconfig/tcedir'
alias cdb='cd /var/lib/boot2docker'

alias l='ls -alrt'
alias h=history

alias d=docker
alias di='docker images'
alias dps='docker ps'
alias dka='docker kill $(docker ps -q --no-trunc)'
alias dsta='docker stop $(docker ps -q --no-trunc)'
alias drmae='docker rm $(docker ps -qa --no-trunc --filter "status=exited" 2>/dev/null) 2>/dev/null'
alias drmiad='docker rmi $(docker images --filter "dangling=true" -q --no-trunc 2>/dev/null) 2>/dev/null'
alias dr='docker run'
alias dv='docker volume'
alias dritrm='docker run -it --rm'
alias dr='docker run -it --rm'
alias dpsa='docker ps -a'
alias din='docker inspect'
drcat() {
	echo "curl -X GET https://kv:5000/v2/_catalog";
	curl -X GET https://kv:5000/v2/_catalog;
}
drtag() {
	echo "curl -X GET https://kv:5000/v2/$1/tags/list;"
	curl -X GET https://kv:5000/v2/$1/tags/list;
}
dvls() { vpath=$(dv inspect -f '{{ .Mountpoint }}' $1); sudo ls -alrth ${vpath}/$2; }
alias dn='docker network'

alias dc='docker run --rm -i -t -v /var/run/docker.sock:/var/run/docker.sock -v `pwd`:`pwd` -w `pwd` docker-compose'

alias gl='git log --oneline --decorate --graph -10 --all --branches'
alias b='./build'
alias r='./run'
alias k='./kill'
alias rba='./run bash'
alias kba='./kill bash'

alias bb='./build blessed'
alias rb='./run blessed'
alias kb='./kill blessed'

alias bs='./build staging'
alias rs='./run staging'
alias ks='./kill staging'

alias be='./build external'
alias re='./run external'
alias ke='./kill external'

alias bi='./build internal'
alias ri='./run internal'
alias ki='./kill internal'

deb() {
	echo "docker exec -u git -it $1 bash";
	docker exec -u git -it $1 bash;
 }
debr() {
	echo "docker exec -u root -it $1 bash";
	docker exec -u root -it $1 bash;
}
debc() {
	echo "docker exec -u git -it $@";
	docker exec -u git -it "$@";
}
debrc() {
	echo "docker exec -u root -it $@";
	docker exec -u root -it "$@";
}
drb() {
	echo "docker run -itd --name $1 busybox";
	docker run -itd --name $1 busybox;
}
scpe() { sudo cp -f "${1}" $(cat $(ls *.path.${2})); sudo chown 105:112 $(cat $(ls *.path.${2}))/"${1}"; }

alias stop='./stop'
alias test='./test'
alias s=stop
alias t=test

alias n='. ./next'
