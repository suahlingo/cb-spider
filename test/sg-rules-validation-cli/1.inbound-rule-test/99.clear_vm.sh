#!/bin/bash

if [ "$1" = "" ]; then
        echo
        echo -e 'usage: '$0' mock|aws|azure|gcp|alibaba|tencent|ibm|openstack|cloudit|ncp|nhncloud'
        echo -e '\n\tex) '$0' aws'
        echo
        exit 0;
fi

echo -e "###########################################################"
echo -e "# Terminate VM "
echo -e "###########################################################"

source ../common/setup.env $1
source setup.env $1

echo -e "# clear nc processes on the client(this node)"
sudo killall nc 2> /dev/null

echo -e "# clear nc processes on the target VM to release local calling process"
ssh -f -i ${KEYPAIR_NAME}.pem -o StrictHostKeyChecking=no cb-user@$P_IP "sudo killall nc"


echo -e "# Try to terminate test VM"
for (( i=1; i <= 30; i++ ))
do
        ret=`../common/7.vm-terminate.sh $1`
        echo -e "$ret"

        result=`echo -e "$ret" |grep "does not exist"`
        if [ "$result" ];then
                break;
        else
                sleep 2
        fi
done

echo -e "\n\n"

echo -e "###########################################################"
echo -e "# Terminated VM "
echo -e "###########################################################"

echo -e "\n\n"

