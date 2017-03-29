#!/bin/bash

#脚本所在的目录
basePath=$(cd "$(dirname "$0")"; pwd)
cd $basePath

err_log=$basePath"/err_log"
record=$basePath"/atlas-record"

yesterdayExtension=`date -d "1 day ago" +".%Y%m%d"`

OIFS="$IFS"
IFS=$'\n'
for host in `ls ../nginx-hosts`; do
    servers+=($(<$host))
done
IFS="$OIFS"

domains=(
   #domains
)

function rsyncLog() {
    for server in ${servers[@]}; do
        rsync -av 10.10.34.18::proxy_nginx_logs/$server/"*"$yesterdayExtension"*" $server"_nginx_logs/" 2>/dev/null
        if [ -d $server"_nginx_logs" ]; then
            cd $server"_nginx_logs"
            for file in `ls`; do
                if [[ "$file" == *.gz ]]; then
                    zcat $file| grep -E -v "checkhealth|favicon|status" >> ../$server".all" 2>/dev/null
                else
                    cat $file| grep -E -v "checkhealth|favicon|status" >> ../$server".all" 2>/dev/null
                fi
                rm $file
            done
            cd ..
            rm -r $server"_nginx_logs"
            cat $server".all" | ./nginx-stat 1> $server".json" 2> $err_log
            rm $server".all"
        fi
    done
}

function stat() {
    for domain in ${domains[@]}; do
        echo >> $record
        echo "---------"$domain"----------" >> $record

        cat *.json | jq -r ".|{\"$domain\"}|.[\"$domain\"].total"|awk '{sum+=$1}END{print "总请求数: "sum}' >> $record
        echo >> $record
        echo "前端状态码统计: " >> $record
        cat *.json | jq -r ".|{\"$domain\"}|.[\"$domain\"].frontEnd.status.counter"|awk 'BEGIN{print "状态码          个数"}{if(NF > 1) sum[$1]+=$2}END{for (i in sum) print i"     "sum[i]}' >> $record
        cat *.json | jq -r ".|{\"$domain\"}|.[\"$domain\"].frontEnd.status.counter"|awk '{if(NF > 1) {sum+=$2; if($1=="\"400\":" || $1=="\"408\":" || $1=="\"500\":" || $1=="\"502\":" || $1=="\"503\":" || $1=="\"504\":") failure+=2}} END{printf"前端失败率: %.4f%%\n\n", failure/sum*100}' 2>/dev/null >> $record

        echo "后端状态码统计: " >> $record
        #如果状态码是5xx。nginx日志会轮循打出各机器的状态码
        cat *.json | jq -r ".|{\"$domain\"}|.[\"$domain\"].backEnd.status.counter"|awk 'BEGIN{print "状态码          个数"}{if(NF > 1) sum[$1]+=$2}END{for (i in sum) print i"     "sum[i]}' >> $record
        cat *.json | jq -r ".|{\"$domain\"}|.[\"$domain\"].backEnd.status.counter"|awk '{if(NF > 1) {sum+=$2; if($1=="\"400\":" || $1=="\"408\":" || $1=="\"500\":" || $1=="\"502\":" || $1=="\"503\":" || $1=="\"504\":") failure+=2}} END{printf"后端失败率: %.4f%%\n\n", failure/sum*100}' 2>/dev/null >> $record

        cat *.json | jq -r ".|{\"$domain\"}|.[\"$domain\"].frontEnd.responseTimeHistogram.counter" | awk 'BEGIN{print "前端响应时间: "}{if(NF > 1) c[$1]+=$2;sum+=$2}END{for (i in c) printf "%s   %d,   %.4f%%\n", i, c[i], c[i]/sum*100}' 2>/dev/null >> $record
        echo >> $record
        cat *.json | jq -r ".|{\"$domain\"}|.[\"$domain\"].backEnd.responseTimeHistogram.counter" | awk 'BEGIN{print "后端响应时间: "}{if(NF > 1) c[$1]+=$2;sum+=$2}END{for (i in c) printf "%s   %d,   %.4f%%\n", i, c[i], c[i]/sum*100}' 2>/dev/null >> $record
    done
}

rsyncLog

stat

content="nginx_log$yesterdayExtension\r\n"
content=`cat $record`
# sent email

rm $record
rm $shanliaoRecord
rm *.json