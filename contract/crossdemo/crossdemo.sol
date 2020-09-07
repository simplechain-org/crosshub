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

    modifier onlyAnchor(uint8 purpose) {
        require(crossChains[purpose].purpose > 0,"purpose err");
        require(crossChains[purpose].anchors[msg.sender].purpose == purpose,"not anchors");
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
    event MakerTx(bytes32 txId, uint256 value, uint256 destValue, string from, string to, uint8 purpose, bytes payload);

    event MakerFinish(bytes32 txId, address to);
    //达成交易 taker
    event TakerTx(bytes32 txId, address from, string to, uint8 chain,bytes payload);

    event AddAnchors(uint8 purpose);

    event RemoveAnchors(uint8 purpose);

    event UpdateRouter(uint8 purpose);

    event AccumulateRewards(uint8 purpose, address anchor, uint256 reward);

    event SetAnchorStatus(uint8 purpose);

    function bitCount(uint64 n) public pure returns(uint64){
        uint64 tmp = n - ((n >>1) &0x36DB6DB6DB6DB6DB) - ((n >>2) &0x9249249249249249);
        return ((tmp + (tmp >>3)) &0x71C71C71C71C71C7) %63;
    }

    //更改跨链交易奖励 管理员操作
    function setReward(uint8 purpose, uint _reward) public onlyOwner { crossChains[purpose].reward = _reward; }

    function getTotalReward(uint8 purpose) public view returns(uint256) { return crossChains[purpose].totalReward; }

    function getChainReward(uint8 purpose) public view returns(uint256) { return crossChains[purpose].reward; }

    function getMaxValue(uint8 purpose) public view returns(uint256) { return crossChains[purpose].maxValue; }

    function accumulateRewards(uint8 purpose, address payable anchor, uint256 reward) public onlyOwner {
        require(reward <= crossChains[purpose].totalReward, "reward err");
        require(crossChains[purpose].anchors[anchor].purpose == purpose, "illegal anchor");
        crossChains[purpose].totalReward = crossChains[purpose].totalReward.safeSub(reward);
        anchor.transfer(reward);
        emit AccumulateRewards(purpose, anchor, reward);
    }

    //登记链信息 管理员操作
    function chainRegister(uint8 purpose,uint256 maxValue, uint8 signConfirmCount, address[] memory _anchors,string memory router) public onlyOwner returns(bool) {
        require (crossChains[purpose].purpose == 0,"purpose err");
        require (_anchors.length <= 64,"_anchors err");
        uint64 temp = 0;
        address[] memory newAnchors;
        address[] memory delAnchors;

        //初始化信息
        crossChains[purpose] = CrossStruct.Chain({
            purpose: purpose,
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
            if (crossChains[purpose].anchors[_anchors[i]].purpose > 0) {
                revert();
            }
            crossChains[purpose].anchorAddress.push(_anchors[i]);
            crossChains[purpose].anchors[_anchors[i]] = CrossStruct.Anchor({purpose:purpose,position:i,status:true,signCount:0,finishCount:0});
        }
        return true;
    }

    //增加锚定矿工，管理员操作
    // position [0, 63]
    function addAnchors(uint8 purpose, address[] memory _anchors) public onlyOwner {
        require (crossChains[purpose].purpose == purpose,"purpose err");
        require (_anchors.length > 0 && _anchors.length < 64,"need _anchors");
        require ((crossChains[purpose].anchorAddress.length + _anchors.length) <= 64,"_anchors err");
        uint64 temp = 0;
        crossChains[purpose].anchorsPositionBit = (temp - 1) >> (64 - crossChains[purpose].anchorAddress.length + _anchors.length);
        //加入锚定矿工
        for (uint8 i=0; i<_anchors.length; i++) {
            if (crossChains[purpose].anchors[_anchors[i]].purpose > 0) {
                revert();
            }
            // 添加的不能是已经删除的
            if (crossChains[purpose].delAnchors[_anchors[i]].purpose > 0){
                revert();
            }

            crossChains[purpose].anchors[_anchors[i]] = CrossStruct.Anchor({purpose:purpose, position:uint8(crossChains[purpose].anchorAddress.length),status:true,signCount:0,finishCount:0});
            crossChains[purpose].anchorAddress.push(_anchors[i]);
        }
        emit AddAnchors(purpose);
    }

    //移除锚定矿工, 管理员操作
    function removeAnchors(uint8 purpose, address[] memory _anchors) public onlyOwner {
        require (crossChains[purpose].purpose > 0,"purpose err");
        require (_anchors.length > 0,"need _anchors");
        require((crossChains[purpose].anchorAddress.length - crossChains[purpose].signConfirmCount) >= _anchors.length,"_anchors err");
        uint64 temp = 0;
        crossChains[purpose].anchorsPositionBit = (temp - 1) >> (64 - crossChains[purpose].anchorAddress.length + _anchors.length);
        for (uint8 i=0; i<_anchors.length; i++) {
            if (crossChains[purpose].anchors[_anchors[i]].purpose > 0) {
                revert();
            }

            uint8 index = crossChains[purpose].anchors[_anchors[i]].position;
            if (index < crossChains[purpose].anchorAddress.length - 1) {
                crossChains[purpose].anchorAddress[index] = crossChains[purpose].anchorAddress[crossChains[purpose].anchorAddress.length - 1];
                crossChains[purpose].anchors[crossChains[purpose].anchorAddress[index]].position = index;
                crossChains[purpose].anchorAddress.pop();
                deleteAnchor(purpose,_anchors[i]);
            } else {
                crossChains[purpose].anchorAddress.pop();
                deleteAnchor(purpose,_anchors[i]);
            }
        }
        emit RemoveAnchors(purpose);
    }

    function deleteAnchor(uint8 purpose,address del) private {
        delete crossChains[purpose].anchors[del];
        // 不能重复删除
        if (crossChains[purpose].delAnchors[del].purpose > 0){
            revert();
        }
        if(crossChains[purpose].delsAddress.length < 64){
            uint64 temp = 0;
            crossChains[purpose].delsPositionBit = (temp - 1) >> (64 - crossChains[purpose].delsAddress.length - 1);
            crossChains[purpose].delAnchors[del] = CrossStruct.Anchor({purpose:purpose, position:uint8(crossChains[purpose].delsAddress.length),status:false,signCount:0,finishCount:0});
            crossChains[purpose].delsAddress.push(del);

        }else{ //bitLen == 64 （处理环）
            delete crossChains[purpose].delAnchors[crossChains[purpose].delsAddress[crossChains[purpose].delId]];
            crossChains[purpose].delsAddress[crossChains[purpose].delId] = del;
            crossChains[purpose].delAnchors[del] = CrossStruct.Anchor({purpose:purpose, position:crossChains[purpose].delId,status:false,signCount:0,finishCount:0});
            crossChains[purpose].delId ++;
            if(crossChains[purpose].delId == 64){
                crossChains[purpose].delId = 0;
            }
        }
    }

    function updateRouter(uint8 purpose, string memory _router) public onlyOwner {
        require (crossChains[purpose].purpose > 0,"purpose err");
        require (keccak256(abi.encodePacked(crossChains[purpose].router)) != keccak256(abi.encodePacked(_router)),"need _anchors");
        crossChains[purpose].router = _router;
        emit UpdateRouter(purpose);
    }

    function setAnchorStatus(uint8 purpose, address _anchor,bool status) public onlyOwner {
        if (!status) {
            uint8 j=0;
            for (uint8 i=0; i<crossChains[purpose].anchorAddress.length; i++) {
                if (crossChains[purpose].anchors[crossChains[purpose].anchorAddress[i]].status) {
                    j++;
                }
            }
            require(j > crossChains[purpose].signConfirmCount);
            crossChains[purpose].anchors[_anchor].status = status; //true Available
            emit SetAnchorStatus(purpose);
        } else {
            crossChains[purpose].anchors[_anchor].status = status; //true Available
            emit SetAnchorStatus(purpose);
        }
    }

    function setSignConfirmCount(uint8 purpose,uint8 count) public onlyOwner {
        require (crossChains[purpose].purpose > 0,"purpose err");
        require (count != 0,"count 0");
        require (count <= crossChains[purpose].anchorAddress.length,"count err");
        crossChains[purpose].signConfirmCount = count;
    }

    function setMaxValue(uint8 purpose,uint256 maxValue) public onlyOwner {
        require (crossChains[purpose].purpose > 0,"purpose err");
        require (maxValue != 0,"maxValue 0");
        require (maxValue > crossChains[purpose].reward,"too less");
        crossChains[purpose].maxValue = maxValue;
    }

    function getMakerTx(bytes32 txId, uint8 purpose) public view returns(uint256){
        return crossChains[purpose].makerTxs[txId].value;
    }

    function getTakerTx(bytes32 txId, address _from, uint8 purpose) public view returns(uint256){
        if (crossChains[purpose].takerTxs[txId].from == _from) {
            return crossChains[purpose].takerTxs[txId].value;
        }
        return 0;
    }

    function getAnchors(uint8 purpose) public view returns(address[] memory _anchors,uint8){
        uint8 j=0;
        for (uint8 i=0; i<crossChains[purpose].anchorAddress.length; i++) {
            if (crossChains[purpose].anchors[crossChains[purpose].anchorAddress[i]].status) {
                j++;
            }
        }
        _anchors = new address[](j);
        uint8 k=0;
        for (uint8 i=0; i<crossChains[purpose].anchorAddress.length; i++) {
            if (crossChains[purpose].anchors[crossChains[purpose].anchorAddress[i]].status) {
                _anchors[k]=crossChains[purpose].anchorAddress[i];
                k++;
            }
        }
        return (_anchors,crossChains[purpose].signConfirmCount);
    }

    function getRouter(uint8 purpose) public view returns(string memory){
        return crossChains[purpose].router;
    }

    function getAnchorWorkCount(uint8 purpose,address _anchor) public view returns (uint256,uint256){
        return (crossChains[purpose].anchors[_anchor].signCount,crossChains[purpose].anchors[_anchor].finishCount);
    }

    function getDelAnchorSignCount(uint8 purpose,address _anchor) public view returns (uint256){
        return (crossChains[purpose].delAnchors[_anchor].signCount);
    }

    //增加跨链交易，arg[1], arg[2]分别为msg.sender在对面链地址和对面链制定接单用户地址
    function makerStart(uint256 destValue, uint8 purpose,string[2] memory arg, bytes memory data) public payable {
        require(msg.value < crossChains[purpose].maxValue,"value err");
        require(crossChains[purpose].purpose > 0,"chainId err"); //是否支持的跨链
        bytes32 txId = keccak256(abi.encodePacked(msg.sender, list(), chainId()));
        assert(crossChains[purpose].makerTxs[txId].value == 0);
        crossChains[purpose].makerTxs[txId] = CrossStruct.MakerInfo({
            value:(msg.value.safeSub(crossChains[purpose].reward)),
            signatureCount:0,
            from:arg[0],
            to:arg[1],
            takerHash:bytes32(0x0) //todo
            });
        crossChains[purpose].totalReward = crossChains[purpose].totalReward.safeAdd(crossChains[purpose].reward);
        emit MakerTx(txId, msg.value, destValue, arg[0], arg[1], purpose, data);
    }

    //锚定节点执行,防作恶
    function makerFinish(CrossStruct.Recept memory rtx) public onlyAnchor(rtx.origin) payable {
        require(crossChains[rtx.origin].anchors[msg.sender].status);
        require(crossChains[rtx.origin].makerTxs[rtx.txId].signatures[msg.sender] != 1);
        require(crossChains[rtx.origin].makerTxs[rtx.txId].value > 0);
        require(keccak256(abi.encodePacked(crossChains[rtx.origin].makerTxs[rtx.txId].from)) == keccak256(abi.encodePacked(rtx.from)),"from err");
        require(keccak256(abi.encodePacked(crossChains[rtx.origin].makerTxs[rtx.txId].to)) == keccak256(abi.encodePacked("")) || keccak256(abi.encodePacked(crossChains[rtx.origin].makerTxs[rtx.txId].to)) == keccak256(abi.encodePacked(rtx.to)) || keccak256(abi.encodePacked(crossChains[rtx.origin].makerTxs[rtx.txId].from)) == keccak256(abi.encodePacked(rtx.to)),"to err");
        require(crossChains[rtx.origin].makerTxs[rtx.txId].takerHash == bytes32(0x0) || crossChains[rtx.origin].makerTxs[rtx.txId].takerHash == rtx.txHash,"txHash err");
        crossChains[rtx.origin].makerTxs[rtx.txId].signatures[msg.sender] = 1;
        crossChains[rtx.origin].makerTxs[rtx.txId].signatureCount ++;
        crossChains[rtx.origin].makerTxs[rtx.txId].to = rtx.to;
        crossChains[rtx.origin].makerTxs[rtx.txId].takerHash = rtx.txHash;
        crossChains[rtx.origin].anchors[msg.sender].finishCount ++;

        if (crossChains[rtx.origin].makerTxs[rtx.txId].signatureCount >= crossChains[rtx.origin].signConfirmCount){
            rtx.taker.transfer(crossChains[rtx.origin].makerTxs[rtx.txId].value);
            delete crossChains[rtx.origin].makerTxs[rtx.txId];
            emit MakerFinish(rtx.txId,rtx.taker);
        }
    }

    function verifySignAndCount(bytes32 hash, uint8 purpose, uint256[] memory v, bytes32[] memory r, bytes32[] memory s) private returns (uint8) {
        uint64 ret = 0;
        uint64 base = 1;
        for (uint i = 0; i < v.length; i++){
            v[i] -= chainId()*2;
            v[i] -= 8;
            address temp = ecrecover(hash, uint8(v[i]), r[i], s[i]);
            if (keccak256(abi.encodePacked(crossChains[purpose].anchors[temp].purpose)) == keccak256(abi.encodePacked(purpose)) && crossChains[purpose].anchors[temp].status){
                crossChains[purpose].anchors[temp].signCount ++;
                ret = ret | (base << crossChains[purpose].anchors[temp].position);
            }
        }
        return uint8(bitCount(ret));
    }

    function verifyOwnerSignAndCount(bytes32 hash, uint8 purpose, uint256[] memory v, bytes32[] memory r, bytes32[] memory s) private returns (uint8) {
        uint64 ret = 0;
        uint64 base = 1;
        uint64 delRet = 0;
        uint64 delBase = 1;
        for (uint i = 0; i < v.length; i++){
            v[i] -= chainId()*2;
            v[i] -= 8;
            address temp = ecrecover(hash, uint8(v[i]), r[i], s[i]);
            if (crossChains[purpose].anchors[temp].purpose == purpose){
                crossChains[purpose].anchors[temp].signCount ++;
                ret = ret | (base << crossChains[purpose].anchors[temp].position);
            }
            if (keccak256(abi.encodePacked(crossChains[purpose].delAnchors[temp].purpose)) == keccak256(abi.encodePacked(purpose))){
                crossChains[purpose].delAnchors[temp].signCount ++;
                delRet = delRet | (delBase << crossChains[purpose].delAnchors[temp].position);
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
        require(crossChains[ctx.origin].takerTxs[ctx.txId].value == 0 || crossChains[ctx.origin].takerTxs[ctx.txId].from != ctx.from,"txId err");
        if(msg.sender == ctx.from){
            require(verifyOwnerSignAndCount(keccak256(abi.encodePacked(ctx.txId, ctx.txHash, ctx.blockHash, ctx.value, ctx.charge, ctx.from, ctx.to, ctx.origin, ctx.purpose, ctx.payload)), ctx.origin, ctx.v,ctx.r,ctx.s) >= crossChains[ctx.origin].signConfirmCount,"sign error");
            crossChains[ctx.origin].takerTxs[ctx.txId] = CrossStruct.TakerInfo({value:ctx.value,from:ctx.from});
            ctx.from.transfer(msg.value);
        } else {
            require(msg.value >= ctx.charge,"price err");
            require(verifySignAndCount(keccak256(abi.encodePacked(ctx.txId, ctx.txHash, ctx.blockHash, ctx.value, ctx.charge, ctx.from, ctx.to, ctx.origin, ctx.purpose, ctx.payload)), ctx.origin, ctx.v,ctx.r,ctx.s) >= crossChains[ctx.origin].signConfirmCount,"sign error");
            crossChains[ctx.origin].takerTxs[ctx.txId] = CrossStruct.TakerInfo({value:ctx.value,from:ctx.from});
            ctx.from.transfer(msg.value);
        }
        emit TakerTx(ctx.txId,ctx.from,to,ctx.origin,data);
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