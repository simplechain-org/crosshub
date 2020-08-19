pragma solidity ^0.6.0;
pragma experimental ABIEncoderV2;
import "./cross_struct.sol";
import "./safe_math.sol";
contract crossDemo{
    //合约管理员
    address public owner;
    using SafeMath for uint256;

    //其他链的信息
    mapping (uint8 => CrossStruct.Chain) public crossChains;

    modifier onlyAnchor(uint8 destination) {
        require(crossChains[destination].destination > 0,"destination err");
        require(crossChains[destination].anchors[msg.sender].destination == destination,"not anchors");
        _;
    }

    modifier onlyOwner() {
        require(msg.sender == owner,"not owner");
        _;
    }

    constructor() public {
        owner = msg.sender;
    }

    //创建交易 maker
    event MakerTx(bytes32 indexed txId, uint256 value, uint256 destValue, string from, string to, uint8 destination, bytes payload);

    event MakerFinish(bytes32 indexed txId, address indexed to);
    //达成交易 taker
    event TakerTx(bytes32 indexed txId, address from, string indexed to, uint8 chain,bytes payload);

    event AddAnchors(uint8 destination);

    event RemoveAnchors(uint8 destination);

    event UpdateRouter(uint8 destination);

    event AccumulateRewards(uint8 destination, address indexed anchor, uint256 reward);

    event SetAnchorStatus(uint8 destination);

    function bitCount(uint64 n) public pure returns(uint64){
        uint64 tmp = n - ((n >>1) &0x36DB6DB6DB6DB6DB) - ((n >>2) &0x9249249249249249);
        return ((tmp + (tmp >>3)) &0x71C71C71C71C71C7) %63;
    }

    //更改跨链交易奖励 管理员操作
    function setReward(uint8 destination, uint _reward) public onlyOwner { crossChains[destination].reward = _reward; }

    function getTotalReward(uint8 destination) public view returns(uint256) { return crossChains[destination].totalReward; }

    function getChainReward(uint8 destination) public view returns(uint256) { return crossChains[destination].reward; }

    function getMaxValue(uint8 destination) public view returns(uint256) { return crossChains[destination].maxValue; }

    function accumulateRewards(uint8 destination, address payable anchor, uint256 reward) public onlyOwner {
        require(reward <= crossChains[destination].totalReward, "reward err");
        require(crossChains[destination].anchors[anchor].destination == destination, "illegal anchor");
        crossChains[destination].totalReward = crossChains[destination].totalReward.safeSub(reward);
        anchor.transfer(reward);
        emit AccumulateRewards(destination, anchor, reward);
    }

    //登记链信息 管理员操作
    function chainRegister(uint8 destination,uint256 maxValue, uint8 signConfirmCount, address[] memory _anchors,string memory router) public onlyOwner returns(bool) {
        require (crossChains[destination].destination > 0,"destination err");
        require (_anchors.length <= 64,"_anchors err");
        uint64 temp = 0;
        address[] memory newAnchors;
        address[] memory delAnchors;

        //初始化信息
        crossChains[destination] = CrossStruct.Chain({
            destination: destination,
            maxValue: maxValue,
            signConfirmCount: signConfirmCount,
            anchorsPositionBit: (temp - 1) >> (64 - _anchors.length),
            anchorAddress:newAnchors,
            reward:0,
            totalReward:0,
            delsPositionBit: (temp - 1) >> 64,
            delsAddress:delAnchors,
            delId:0,
            router:router
            });

        //加入锚定矿工
        for (uint8 i=0; i<_anchors.length; i++) {
            if (crossChains[destination].anchors[_anchors[i]].destination > 0) {
                revert();
            }
            crossChains[destination].anchorAddress.push(_anchors[i]);
            crossChains[destination].anchors[_anchors[i]] = CrossStruct.Anchor({destination:destination,position:i,status:true,signCount:0,finishCount:0});
        }
        return true;
    }

    //增加锚定矿工，管理员操作
    // position [0, 63]
    function addAnchors(uint8 destination, address[] memory _anchors) public onlyOwner {
        require (crossChains[destination].destination == destination,"destination err");
        require (_anchors.length > 0 && _anchors.length < 64,"need _anchors");
        require ((crossChains[destination].anchorAddress.length + _anchors.length) <= 64,"_anchors err");
        uint64 temp = 0;
        crossChains[destination].anchorsPositionBit = (temp - 1) >> (64 - crossChains[destination].anchorAddress.length + _anchors.length);
        //加入锚定矿工
        for (uint8 i=0; i<_anchors.length; i++) {
            if (crossChains[destination].anchors[_anchors[i]].destination > 0) {
                revert();
            }
            // 添加的不能是已经删除的
            if (crossChains[destination].delAnchors[_anchors[i]].destination > 0){
                revert();
            }

            crossChains[destination].anchors[_anchors[i]] = CrossStruct.Anchor({destination:destination, position:uint8(crossChains[destination].anchorAddress.length),status:true,signCount:0,finishCount:0});
            crossChains[destination].anchorAddress.push(_anchors[i]);
        }
        emit AddAnchors(destination);
    }

    //移除锚定矿工, 管理员操作
    function removeAnchors(uint8 destination, address[] memory _anchors) public onlyOwner {
        require (crossChains[destination].destination > 0,"destination err");
        require (_anchors.length > 0,"need _anchors");
        require((crossChains[destination].anchorAddress.length - crossChains[destination].signConfirmCount) >= _anchors.length,"_anchors err");
        uint64 temp = 0;
        crossChains[destination].anchorsPositionBit = (temp - 1) >> (64 - crossChains[destination].anchorAddress.length + _anchors.length);
        for (uint8 i=0; i<_anchors.length; i++) {
            if (crossChains[destination].anchors[_anchors[i]].destination > 0) {
                revert();
            }

            uint8 index = crossChains[destination].anchors[_anchors[i]].position;
            if (index < crossChains[destination].anchorAddress.length - 1) {
                crossChains[destination].anchorAddress[index] = crossChains[destination].anchorAddress[crossChains[destination].anchorAddress.length - 1];
                crossChains[destination].anchors[crossChains[destination].anchorAddress[index]].position = index;
                crossChains[destination].anchorAddress.pop();
                deleteAnchor(destination,_anchors[i]);
            } else {
                crossChains[destination].anchorAddress.pop();
                deleteAnchor(destination,_anchors[i]);
            }
        }
        emit RemoveAnchors(destination);
    }

    function deleteAnchor(uint8 destination,address del) private {
        delete crossChains[destination].anchors[del];
        // 不能重复删除
        if (crossChains[destination].delAnchors[del].destination > 0){
            revert();
        }
        if(crossChains[destination].delsAddress.length < 64){
            uint64 temp = 0;
            crossChains[destination].delsPositionBit = (temp - 1) >> (64 - crossChains[destination].delsAddress.length - 1);
            crossChains[destination].delAnchors[del] = CrossStruct.Anchor({destination:destination, position:uint8(crossChains[destination].delsAddress.length),status:false,signCount:0,finishCount:0});
            crossChains[destination].delsAddress.push(del);

        }else{ //bitLen == 64 （处理环）
            delete crossChains[destination].delAnchors[crossChains[destination].delsAddress[crossChains[destination].delId]];
            crossChains[destination].delsAddress[crossChains[destination].delId] = del;
            crossChains[destination].delAnchors[del] = CrossStruct.Anchor({destination:destination, position:crossChains[destination].delId,status:false,signCount:0,finishCount:0});
            crossChains[destination].delId ++;
            if(crossChains[destination].delId == 64){
                crossChains[destination].delId = 0;
            }
        }
    }

    function updateRouter(uint8 destination, string memory _router) public onlyOwner {
        require (crossChains[destination].destination > 0,"destination err");
        require (keccak256(abi.encodePacked(crossChains[destination].router)) != keccak256(abi.encodePacked(_router)),"need _anchors");
        crossChains[destination].router = _router;
        emit UpdateRouter(destination);
    }

    function setAnchorStatus(uint8 destination, address _anchor,bool status) public onlyOwner {
        if (!status) {
            uint8 j=0;
            for (uint8 i=0; i<crossChains[destination].anchorAddress.length; i++) {
                if (crossChains[destination].anchors[crossChains[destination].anchorAddress[i]].status) {
                    j++;
                }
            }
            require(j > crossChains[destination].signConfirmCount);
            crossChains[destination].anchors[_anchor].status = status; //true Available
            emit SetAnchorStatus(destination);
        } else {
            crossChains[destination].anchors[_anchor].status = status; //true Available
            emit SetAnchorStatus(destination);
        }
    }

    function setSignConfirmCount(uint8 destination,uint8 count) public onlyOwner {
        require (crossChains[destination].destination > 0,"destination err");
        require (count != 0,"count 0");
        require (count <= crossChains[destination].anchorAddress.length,"count err");
        crossChains[destination].signConfirmCount = count;
    }

    function setMaxValue(uint8 destination,uint256 maxValue) public onlyOwner {
        require (crossChains[destination].destination > 0,"destination err");
        require (maxValue != 0,"maxValue 0");
        require (maxValue > crossChains[destination].reward,"too less");
        crossChains[destination].maxValue = maxValue;
    }

    function getMakerTx(bytes32 txId, uint8 destination) public view returns(uint256){
        return crossChains[destination].makerTxs[txId].value;
    }

    function getTakerTx(bytes32 txId, address _from, uint8 destination) public view returns(uint256){
        if (crossChains[destination].takerTxs[txId].from == _from) {
            return crossChains[destination].takerTxs[txId].value;
        }
        return 0;
    }

    function getAnchors(uint8 destination) public view returns(address[] memory _anchors,uint8){
        uint8 j=0;
        for (uint8 i=0; i<crossChains[destination].anchorAddress.length; i++) {
            if (crossChains[destination].anchors[crossChains[destination].anchorAddress[i]].status) {
                j++;
            }
        }
        _anchors = new address[](j);
        uint8 k=0;
        for (uint8 i=0; i<crossChains[destination].anchorAddress.length; i++) {
            if (crossChains[destination].anchors[crossChains[destination].anchorAddress[i]].status) {
                _anchors[k]=crossChains[destination].anchorAddress[i];
                k++;
            }
        }
        return (_anchors,crossChains[destination].signConfirmCount);
    }

    function getRouter(uint8 destination) public view returns(string memory){
        return crossChains[destination].router;
    }

    function getAnchorWorkCount(uint8 destination,address _anchor) public view returns (uint256,uint256){
        return (crossChains[destination].anchors[_anchor].signCount,crossChains[destination].anchors[_anchor].finishCount);
    }

    function getDelAnchorSignCount(uint8 destination,address _anchor) public view returns (uint256){
        return (crossChains[destination].delAnchors[_anchor].signCount);
    }

    //增加跨链交易，arg[1], arg[2]分别为msg.sender在对面链地址和对面链制定接单用户地址
    function makerStart(uint256 destValue, uint8 destination,string[2] memory arg, bytes memory data) public payable {
        require(msg.value < crossChains[destination].maxValue,"value err");
        require(crossChains[destination].destination > 0,"chainId err"); //是否支持的跨链
        bytes32 txId = keccak256(abi.encodePacked(msg.sender, list(), chainId()));
        assert(crossChains[destination].makerTxs[txId].value == 0);
        crossChains[destination].makerTxs[txId] = CrossStruct.MakerInfo({
            value:(msg.value.safeSub(crossChains[destination].reward)),
            signatureCount:0,
            from:arg[0],
            to:arg[1],
            takerHash:bytes32(0x0) //todo
            });
        crossChains[destination].totalReward = crossChains[destination].totalReward.safeAdd(crossChains[destination].reward);
        emit MakerTx(txId, msg.value, destValue, arg[0], arg[1], destination, data);
    }

    //锚定节点执行,防作恶
    function makerFinish(CrossStruct.Recept memory rtx) public onlyAnchor(rtx.destination) payable {
        require(crossChains[rtx.destination].anchors[msg.sender].status);
        require(crossChains[rtx.destination].makerTxs[rtx.txId].signatures[msg.sender] != 1);
        require(crossChains[rtx.destination].makerTxs[rtx.txId].value > 0);
        require(keccak256(abi.encodePacked(crossChains[rtx.destination].makerTxs[rtx.txId].from)) == keccak256(abi.encodePacked(rtx.from)),"from err");
        require(keccak256(abi.encodePacked(crossChains[rtx.destination].makerTxs[rtx.txId].to)) == keccak256(abi.encodePacked("")) || keccak256(abi.encodePacked(crossChains[rtx.destination].makerTxs[rtx.txId].to)) == keccak256(abi.encodePacked(rtx.to)) || keccak256(abi.encodePacked(crossChains[rtx.destination].makerTxs[rtx.txId].from)) == keccak256(abi.encodePacked(rtx.to)),"to err");
        require(crossChains[rtx.destination].makerTxs[rtx.txId].takerHash == bytes32(0x0) || crossChains[rtx.destination].makerTxs[rtx.txId].takerHash == rtx.txHash,"txHash err");
        crossChains[rtx.destination].makerTxs[rtx.txId].signatures[msg.sender] = 1;
        crossChains[rtx.destination].makerTxs[rtx.txId].signatureCount ++;
        crossChains[rtx.destination].makerTxs[rtx.txId].to = rtx.to;
        crossChains[rtx.destination].makerTxs[rtx.txId].takerHash = rtx.txHash;
        crossChains[rtx.destination].anchors[msg.sender].finishCount ++;

        if (crossChains[rtx.destination].makerTxs[rtx.txId].signatureCount >= crossChains[rtx.destination].signConfirmCount){
            rtx.taker.transfer(crossChains[rtx.destination].makerTxs[rtx.txId].value);
            delete crossChains[rtx.destination].makerTxs[rtx.txId];
            emit MakerFinish(rtx.txId,rtx.taker);
        }
    }

    function verifySignAndCount(bytes32 hash, uint8 destination, uint256 remoteChainId, uint256[] memory v, bytes32[] memory r, bytes32[] memory s) private returns (uint8) {
        uint64 ret = 0;
        uint64 base = 1;
        for (uint i = 0; i < v.length; i++){
            v[i] -= remoteChainId*2;
            v[i] -= 8;
            address temp = ecrecover(hash, uint8(v[i]), r[i], s[i]);
            if (keccak256(abi.encodePacked(crossChains[destination].anchors[temp].destination)) == keccak256(abi.encodePacked(destination)) && crossChains[destination].anchors[temp].status){
                crossChains[destination].anchors[temp].signCount ++;
                ret = ret | (base << crossChains[destination].anchors[temp].position);
            }
        }
        return uint8(bitCount(ret));
    }

    function verifyOwnerSignAndCount(bytes32 hash, uint8 destination, uint256 remoteChainId, uint256[] memory v, bytes32[] memory r, bytes32[] memory s) private returns (uint8) {
        uint64 ret = 0;
        uint64 base = 1;
        uint64 delRet = 0;
        uint64 delBase = 1;
        for (uint i = 0; i < v.length; i++){
            v[i] -= remoteChainId*2;
            v[i] -= 8;
            address temp = ecrecover(hash, uint8(v[i]), r[i], s[i]);
            if (crossChains[destination].anchors[temp].destination == destination){
                crossChains[destination].anchors[temp].signCount ++;
                ret = ret | (base << crossChains[destination].anchors[temp].position);
            }
            if (keccak256(abi.encodePacked(crossChains[destination].delAnchors[temp].destination)) == keccak256(abi.encodePacked(destination))){
                crossChains[destination].delAnchors[temp].signCount ++;
                delRet = delRet | (delBase << crossChains[destination].delAnchors[temp].position);
            }
        }
        return uint8(bitCount(ret)+bitCount(delRet));
    }

    //to是msg.sender在对面链的地址
    //todo data编解码
    function taker(CrossStruct.Order memory ctx,string memory to,bytes memory data) payable public{
        require(ctx.v.length == ctx.r.length,"vrs err");
        require(ctx.v.length == ctx.s.length,"vrs err");
        require(ctx.to == address(0x0) || ctx.to == msg.sender || ctx.from == msg.sender,"to err");
        require(crossChains[ctx.chain].takerTxs[ctx.txId].value == 0 || crossChains[ctx.chain].takerTxs[ctx.txId].from != ctx.from,"txId err");
        if(msg.sender == ctx.from){
            require(verifyOwnerSignAndCount(keccak256(abi.encodePacked(ctx.txId, ctx.txHash, ctx.blockHash, ctx.value, ctx.destinationValue, ctx.from, ctx.to, ctx.chain, ctx.destination, ctx.payload)), ctx.chain,chainId(),ctx.v,ctx.r,ctx.s) >= crossChains[ctx.chain].signConfirmCount,"sign error");
            crossChains[ctx.chain].takerTxs[ctx.txId] = CrossStruct.TakerInfo({value:ctx.value,from:ctx.from});
            ctx.from.transfer(msg.value);
        } else {
            require(msg.value >= ctx.destinationValue,"price err");
            require(verifySignAndCount(keccak256(abi.encodePacked(ctx.txId, ctx.txHash, ctx.blockHash, ctx.value, ctx.destinationValue, ctx.from, ctx.to, ctx.chain, ctx.destination, ctx.payload)), ctx.chain,chainId(),ctx.v,ctx.r,ctx.s) >= crossChains[ctx.chain].signConfirmCount,"sign error");
            crossChains[ctx.chain].takerTxs[ctx.txId] = CrossStruct.TakerInfo({value:ctx.value,from:ctx.from});
            ctx.from.transfer(msg.value);
        }
        emit TakerTx(ctx.txId,ctx.from,to,ctx.chain,data);
    }

    function chainId() public pure returns (uint256 id) {
        assembly {
            id := chainid()
        }
    }

    function list() public pure returns (uint256 ll) {
        assembly {
            ll := nonce()
        }
    }
}