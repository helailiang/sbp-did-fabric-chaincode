export CH_NAME="hll8"export ccName="did"peer lifecycle chaincode package $ccName.tar.gz --path src/did-fabric-contract --lang golang --label $ccNamepeer lifecycle chaincode install $ccName.tar.gz
查看安装结果：
peer lifecycle chaincode queryinstalled

export ORDERER_CONTAINER="order1.ordernode.private.bsnbase.com:17051"
export TLS_ROOT_CA="$PWD/certs/ordererOrganizations/ordernode.private.bsnbase.com/orderers/order1.ordernode.private.bsnbase.com/tls/ca.crt"
peer lifecycle chaincode approveformyorg -o $ORDERER_CONTAINER  --tls true --cafile $TLS_ROOT_CA --channelID $CH_NAME --name $ccName --init-required --package-id $ccName:f17add3d723f34f6244163d9fdfe7141c6fa6fe44e4b1d1db445c4f8bdf5750f --sequence 3 --waitForEvent --version "3"


peer lifecycle chaincode commit -o $ORDERER_CONTAINER --tls true --cafile $TLS_ROOT_CA --channelID $CH_NAME --name $ccName --sequence 3 --init-required  --version 3

peer chaincode invoke  -C $CH_NAME -n $ccName --isInit -c '{"Args":["permission:InitProject","bsn","true","true","true","true","server001","pro001"]}' -o $ORDERER_CONTAINER --tls true --cafile $TLS_ROOT_CA 


admin
PL,OKM09*
