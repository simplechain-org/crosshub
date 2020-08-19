pragma solidity ^0.6.0;
library CrossStruct {
    //仅做信息登记，关联chainId
    struct Chain{
        uint8 destination;
        uint8 signConfirmCount;//最少签名数量
        uint256 maxValue;
        uint64 anchorsPositionBit;// 锚定节点 二进制表示 例如 1101001010, 最多62个锚定节点，空余位置0由外部计算
        address[] anchorAddress;
        mapping(address=>Anchor) anchors;   //锚定矿工列表 address => Anchor
        mapping(bytes32=>MakerInfo) makerTxs; //挂单 交易完成后删除交易，通过发送日志方式来呈现交易历史。
        mapping(bytes32=>TakerInfo) takerTxs; //跨链交易列表 吃单 hash => Transaction[]
        mapping(address=>Anchor) delAnchors; //删除锚定矿工列表 address => Anchor
        uint64 delsPositionBit;
        address[] delsAddress;
        uint8 delId;
        uint256 reward;
        uint256 totalReward;
        string router;     //对面链路由节点
    }

    struct Anchor {
        uint8 destination;
        uint8 position; // anchorsPositionBit
        bool status;//true Available
        uint256 signCount;
        uint256 finishCount;
    }

    struct MakerInfo {
        uint256 value;
        uint8 signatureCount;
        mapping (address => uint8) signatures;
        string from;
        string to;
        bytes32 takerHash;
    }

    struct TakerInfo {
        uint256 value;
        address payable from;
    }

    struct Order {
        bytes32 txId;
        bytes32 txHash;
        bytes32 blockHash;
        uint256 value;
        uint256 destinationValue;
        address payable from;
        address to;
        uint8 chain;
        uint8 destination;
        bytes payload;
        uint256[] v;
        bytes32[] r;
        bytes32[] s;
    }

    struct Recept {
        bytes32 txId;
        bytes32 txHash;
        string from;
        string to;
        address payable taker;
        uint8 chain;
        uint8 destination;
        bytes data;
    }
}