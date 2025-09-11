package db

import (
	"reflect"

	doc "github.com/aergoio/aergo-indexer-2.0/indexer/documents"
)

var checkHashList = []string{
	//"DGZBxpoAgdSkhKPccYNBu5ZemzX791xscbBo4oZDGvdQ",
	//"BsDGVkuUSQCa1SJqhUEqc4huB2ZMpbs5CfkHWNEUoY8e",
	//"6ynPZpcpVf3HvPoKKJdxDXD78h88SELDjJA3C1Lc9rao",
	//"GjpjWh25AisqjDUkJqE4HwxBp3j8A6qNwZPj3BRCREXR",
	//"CaxjcxVznFiv2spUhSjxre21dE6Tfn99gtEMDH3dhrTh",
	//"7qMvGXkx3JE9Gt4kqi7GSZsm1zAPgz9Ezza9njxQEAcz",
	//"CCW2XpsPrr2DPVeJLm5r2AkDPu7VfSBkPXdirzR83fXV",
	//"6z1ewUsEroV5C1rs5BPBSFhxFjowh9WM3vGfHySvziwt",
	//"9z75sjcTdEtz869saQK5jZHyeKBoc73ejqnCLZjMKMgT",
	//"22iHRjQhVDoBo85nivN4i1ZnMvGjcsJeLmEEQWZr1ACc",
	//"26KGoXWpZHQdbcZdoHL1gUfBetSLCC8LH26hJaTtueg5",
	//"5AeqnjJBgeWStU8ebnWndhPESpcUvqR71JEFp1GtfvgX",
	//"HZYd1v7obPCwP7wzkwWGRyi6kjr49rgzaDWyKZDnnhmR",
	//"8bJDcYYDthmebKomYLpx8H37DFoXs1u7nCFHsNMCaGwn",
	//"ETySjTa3HSzevftVsw3DCu7AQE9Ad4hEqBpacdiXGAkU",
	//"3mABvtnTMFVtvz8ZNa7b7YXgVZnjFFTYZTjJRFT6mdkS",
	//"6bSqPDw55492wEEShpp5mTkwnKTVub2qoPRYvxuUBSNh",
	//"DjcJgVbpRqA5VtQZooYQqDUztXPxRq83U9ibgvzYo26c",
	//"GKgA8HCadA68pH7t5DEeGB3eH41NwZsgLtpFLHkvAvyr",
	//"9z7EcqghhPx4NzGsbSVgnStHhTGgYLVKK9aTwrj5rxK",
	//"E9NpJimUMkiECWqK7ciuU4EKHfDpCRufoFSLK3qsLgry",
	//"BqLWCvVj7iMnCg8ZpE2znWW7vhnZhQ7JzgbKmzm6Xz7r",
	//"HsAUcpmb4q8RkZBzCsn8unLCn8saHJ7rzm7ugoMmDQMD",
	//"HwRuEm5qTYTKCxT57jaNWws59ZpTQc3QVioX9E47mVPr",
	//"B8r1mSaQJczNiiixhsrJeUoWyus4SNL3tTFmKTPmEa18",
	//"CcwPoWT6eQqPUAHEzerHx2dzbkaQwkXQo2tixAFzufcb",
	//"CbMQkB29gR1Z5Ao7nLhSoH9P4yLU5TQY8tTmnAKPCnVq",
	//"533oRHAJnZKtkmu7MnSbnoLAT1KqeEMREm4Nkz1t54Rq",
	//"C5mgemJbiuEqz77NKX1nV2nxGWDED7aXeE6P4aJbAdDG",
	//"9tLaegR1qsrSqDVWL6VbdCi13DYsuRpKSdTw47UR6R7g",
	//"GLcHmyUxRY7DCfp2QnGRC93aE6tDUGH4Y3YqUDDhemWR",
	//"25mBEfyJeoU6eiKm8Krg7bEtZtnVU5MsrxtAWZNxzHxF",
	//"FBhB2WykKERLr2HCZdPQjUB2foYq12BYVpVBVxM13h9U",
	//"JBcEZbqznDTN4ntzZP2URT1JUrE9Wv7mwPJTy69Ns2nX",
	//"DdqdvnmD4BepFhxme2m4XPaahX2DLxhvrJGs3BEXVdjw",
	//"FLATPMkcc2Fsmv5zv8bLVb2FUijMiaTfPfRvUyoxBjE9",
	//"B3RhTHQUXos8KZuqSsTUkwYZZSkxorth5NV3rDS13Q31",
	//"5ND94NnPFvWJ2ppLRm9DDWLaB4bfE17JZf6o99KRy3V5",
	//"A7z9CQzuwihJXoz7B67vNJFm36UZx3zNE17V8ye3pNbn",
	//"Cv4SgvJNzuCcYfzTmWHt5EfpQjuHm3gg2gn7YHqtvxsp",
	//"CXJZWHFccoMESCQJTv8dehhKdRCBCYUQ9teJe6DfGJCm",
	//"Fec4pE974umGSZfVcfysrvi6PkQDy49BYb5VQYTEACVM",
	//"AiZEEmbaPoYGC5hZxD9DfaxZNNEnHtPwBDqcLqi97sPE",
	//"6jWkRWLFEEnHbJPWy6hkPUEuqDpDZuZpDF5dE1W5Gc8j",
	//"uX9uQodb4R7ECS8fnmgGMRJPHtbGgxYt2ZqZda4fUsG",
	//"75etfv2egwk7Ff2mpJv7fFJaEY6A9mQouJwVmR2SKfHu",
	//"4Wq3qDHXvfNT9sxPLynPr9gUTNU6knrFbHHoAsyH9oRn",
	//"2SboqGf1g6nNsXbvBen3YQfWA2Xfd27HMiM8uVat5vFS",
	//"tZQPf1X2hT9SYzutyDsYUcmJ76bE7BgmfUgNh3JoMNU",
	//"6ZeL4bvgVyaRAvBHgGYgKchLnKAies4GvvdSAVL8H7X4",
	//"FmVYZUHPhDYxtb1sPtrXPNTATBTskX5KihH2zS6FGUTA",
	//"6aicekmZoS8P6btLJFqSHUQqp8KG7zN6xEVj9zw7pR56",
	//"J6vgoa7PPwXSQJEq82cjie1fbvHd6JDZmpTVKRLxY1oo",
	//"81t57ky4sfxy2RaGgoCRcjPeB7kq4YV7aUDLLDUibjZg",
	//"3waiLyDSPTDpmJUW7Am9Cpi69F7GVnScnfUdZ6Ua7cDt",
	//"3nkV5cGJa6TPndw8m6bjuMoVBuDD6McEGBVg9xtG5CS2",
	//"8dKaGEiSoHvkCfJMv5f1yVSHpwAEwga7bX4wnAdqi4Se",
	//"4yrvXE2FLWn3SyGGVhCH5ogVqX9ipQRqKu3NniHecV4a",
	//"FD3we88df82jYSbGAiNUiLWa4ihVozux1kPXkf55cRTM",
	//"EbLvrR1E4nZTiRUxCxhtKyjXQuDCahcxoYWy6KrnbwkQ",
	//"AWqokqEQoGPTXCRPd7UzwNKNrfWBKFENscetJNHEviBb",
	//"4f7FRSwMTwMCXJsCW3DPGRTVCZdvGzFEZW4GYKLeVXWJ",
	//"7QHfYf6FMr4mRVvRRdHhGtjtmqrSyT9KN5Lng7zavk7z",
	//"F9PJR42QtVjNgwQzffeDXGNALYd4NMzFSKGKTd6JJSyq",
	//"4ztxha5cb3R1M2atAmN1aBaNZosXJspQkDWtmFxCacip",
	//"2saDEQmu3hZEABPiUr7HyPLdDtMM7CuPTQ9b2Z7bKTUv",
	//"2w8SRtErYNQdWPKbM44jDnig12mub3JhzTuLBtYQTm4w",
	//"6jHNeN921hEEANzYo5AK66P5P1cGZyPVZufu1erYcajB",
	//"6btNFwYT1Mx8NmxvaQQ7UrXy3JtC9UUTciFtc1gY7aGF",
	//"AhaNqnEqGygJukAaFhkvA7r7xoAGqkUyVZhVBkL4U67Z",
	//"6MTgeBmoQa6KL57Y8B2M3e1VUKS9yMmjtpTesGAX3Xks",
	//"3uQHDR8sLEUf2K7SsP9Z7CnTq4y3BzznX1PbycH2x3SS",
	//"AKWhmJckNJ4V2Dbx2upph8H4JjjVVoLP1tebsnXnhYM8",
	//"CjqMWqogds6kLyBZDXzWJv9Go1R8kURidRvJjfkLjaVD",
	//"ADwfL84ouTvHpTu23QSwgDVrEYUWgaDkzTVAHsfD1R4W",
	//"3s17Y8gMATGgypeMUsyDbXfd3nW2keZN8cUuqTnGU1Pn",
	//"8BGawjv1744VnugY3SEUVTzVLjLfAyYsuH6gjKa6svgv",
	//"AjUWcWpvPn6JmFWT66DX6U41abkSajdUt8h2cBKpHLuc",
	//"26GUuyh1JnLXRdVERY9Fc7UPBk91ZT2v1gjRVFBiU62S",
	//"HPxDDTXAniQo3uBzg9csL7bD5XxTUfKLmCWipQVEq6vZ",
	//"HEsJw95UwaPDUYV2NHvxEZk1qMkV2FfTduGZ7WDbshoU",
	//"4qiTicMtBP6xS3x1rFd8Z7siz7V5V2DKRJWky6ffHp5n",
	//"HgU8FB9YjZLxADegfYyJAVvU4B8ALmjpTtB4tGte2W2A",
	//"4cY9AXUxz8dLuTEvFmXHyp6Dr8S4xdw6JfsTwe95XZnZ",
	//"E1h17FHQkoUgrnEGJzUGpiVWG7j5Ebd1eCy53b9MnQYh",
	//"69LVpY2DDr4hTryRnNmzhtaZkHeuZW42nKBzdMkmPCrr",
	//"HyYzyhV4HWa3z9irY9nEqiAj6WPYj2e7UKuentVELYCi",
	//"CGEr9hAqeJRSX5zqrQuYD183kaQEJN4gr7fiQp9SRvxr",
	//"APZ2avMbgrBU2LPQqtGFXuWHtmYPSNbBkPEegwFqaozK",
	//"8wyTRZotWe5d6w6bhF1ovR2AD4RhNgGys9PQAM72kdjB",
	//"HestcRpWFuaXoexHVX7k1WuXhHEJEaR13YFtaTMXFS9v",
	//"B8UFKEcffpxTJmhK7A2mvzzy4J5oa3ruTxnzpZuvZV5n",
	//"2TXaRLqcZDaGZQ3iwZMNukpGQapDWq1rAjDxaNzZ11nk",
	//"F8EVNn5kUem45BYqbRe42duJ74Uyqx52FvgRpWPVLKEP",
	//"5CDYEipjvJcRMLVnuAGeftAWThg1sFsRU6fMbQDAQ2cT",
	//"FSUXRcPeoaDMg1TTdXTmA5K9gRdGfwBsqaKMey4YUjuq",

	// in block 152388001
	"tZQ12xKRJLzbFpop6btzHA79SVyNMt3aAcmmwQERJys",
	"4ThxEZJj41Ev8QJBLXjfZ873mstJsWwYDiH19hpCnLgq",
	"HmZEYqLfrS7ofYwF9dRv7fNU1CA1xEFtsC5BNY16okmr",
	"2z8rueP11oSjsHJ4ZDEkyPHtmDpZQLg5TQiXZQb6DSDm",
	"FviXF7kYBHEQfjXuC2Nw8c9xVJ8aFmAJDPfohe1j8Uxp",
	"HwA4688wumfgQDbhb9hoyfVMGB8NaFJBUBm8BagxyoRY",
	"CkhnobbZWTnRxagb5GuFSF5Sg1j7t7yi8pPwGRo4oVHg",
	"8LGkmygh4j1hUXi1hFBX3s2S4Lpda1BF334jdTfBrLeo",
	"HNn9Qnz2wCHFewWhTfQHLN28TnGnSka7NaJDFR8eKmKR",
	"7um32RzsyBjaTku9h8LS4m4fbTDYtAioFM8MpH46R3DP",
	"C9qYdBtMASLB1psnM4yNyv43yNssstS9pVUqcAce8Vd5",
	"5ksuFCGBZnBjHtTU2g4kYXAVMWFBAkB6UmXSD8QnY5Th",
	"EZxAnVC6v7Cp9bWUMqj2XW4xPSujkN14kFh9yeHuX7CK",
	"DZTzBZKuojrfvnAnBv2xDPc5NUtbKuG39jLsQawSbvxk",
	"Br7dcXt7g3TmsV1nLpdjdC8NhzEtcHAk81mEHYKQoiDS",
	"AyR8Mmbw7kdsWQ48GX5UJCf9VC98mA8GxsuFe31TuvFg",
	"D61rswZYuFvXrxy1rsPk2DKhPei1HZNb1gvh3NZNjkr",
	"LNKvpYuPujtG5KDKM9ANPt4g1DttM849AjVpXYSmwVk",
	"B3BneAejBDhri5Kdw4kykc3bp5JiwrNXBK3J2JiqykPi",
	"GVJmkS2m4UNJSdgVAnXQXVYtpLbm1TSm6NNsBZc4a32T",
	"GWEUyVaV5gkUyMbYuNfpMqzF7rQHzxn3ZoPrp9EfiwCa",
	"HwdjmNzLXur1utZBTk73MhjY7RcTSyeSPGRRxFPDBg7U",
	"F3a6ZhpYoXfWfcpNM5q9ihdJk7Sk8ZQ5Zsh4AAjKhZzy",
	"J9Q6U965111inoH1tKW2M9Xfs6VWHxxVTu3eJ7c6Y78X",
	"Am77zF5RLJPJp1NX5PvquMUBnqTwSKLAk4HEiDtQmvgD",
	"J3oUdnz3Y3MiJBTgskR9JhsxVMAyEHV9vckEnXP4izEA",
	"2JJcyX3Hq9bQE5SnbJNb2CTChgsPRhXfbzdTzrZDN4Ex",
	"5PSFogNpxdzXLRqoFN1j1wMtLM3jai6PjREgqUzNeNYU",
	"3XHRqQPwXJVkimXMM5KPpevNmYoGSnzAfFGvUi9TrKET",
	"2Sm3jEbSDnuw7EMMWU2cWcARdvgeA8UT9JhkcgKtJYui",
	"8R4hQLjy4sXiXgF7kwAgozvwxS7ouDBQwWaZP2ESGd9w",
	"Ex16REpXwey8X5TsJPorjnm1w9cUgtKye9ejuKZERNv9",
	"CrZ9MserjF2KvL6sdDe4ZK1JiMWbgj9Csr1totS8VwrD",
	"GHDAEheySLc8dsu9ceGah5aqKdnAPvgP2E3iWGCxnLFH",
	"9ickpsda4JYXVVSvq8aFPSyPeVxnAvJCtssPWvVw7Ubv",
	"7JjUjdjoav4QiXrmCLaZM9doz8XPQnuy7WektTKYtvfG",
	"HMfk5Xno6x3S5CP6cZV3e6z9MHNRLmVpv4zFqgKpTqzy",
}

var toCheckTx map[string]bool = make(map[string]bool)

func init() {
	for _, h := range checkHashList {
		toCheckTx[h] = true
	}
}

func traceESWriteTx(indexName string, document doc.DocType, action string) {
	switch value := document.(type) {
	case *doc.EsTx:
		if toLog(value.Id) {
			logger.Debug().Str("action", action).Str("indexName", indexName).Str("hash", value.Id).Msg("Insert TX")
		}
	case *doc.EsContractCall:
		if toLog(value.Id) {
			logger.Debug().Str("action", action).Str("indexName", indexName).Str("func", value.Function).Str("TxHash", value.TxHash).Msg("Insert ContractCall")
		}
	case *doc.EsInternalOperations:
		if toLog(value.Id) {
			logger.Debug().Str("action", action).Str("indexName", indexName).Str("internalOp", value.Operations).Str("TxHash", value.TxId).Msg("Insert InternalOperations")
		}
	case *doc.EsNFT:
		logger.Debug().Str("action", action).Str("indexName", indexName).Str("NFT", value.Id).Uint64("inBlock", value.BlockNo).Msg("Insert NFT")
	case *doc.EsToken, *doc.EsBlock:
		// do nothing
	default:
		docDataType := reflect.TypeOf(document)
		logger.Debug().Str("action", action).Str("indexName", indexName).Str("typeName", docDataType.String()).Msg("Insert")

	}
}

func toLog(value string) bool {
	// 테스트를 위해 일단 열어둠
	return true
}
