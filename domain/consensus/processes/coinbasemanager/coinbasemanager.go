package coinbasemanager

import (
	"math"

	"github.com/bugnanetwork/bugnad/domain/consensus/model"
	"github.com/bugnanetwork/bugnad/domain/consensus/model/externalapi"
	"github.com/bugnanetwork/bugnad/domain/consensus/utils/constants"
	"github.com/bugnanetwork/bugnad/domain/consensus/utils/hashset"
	"github.com/bugnanetwork/bugnad/domain/consensus/utils/subnetworks"
	"github.com/bugnanetwork/bugnad/domain/consensus/utils/transactionhelper"
	"github.com/bugnanetwork/bugnad/infrastructure/db/database"
	"github.com/pkg/errors"
)

type coinbaseManager struct {
	subsidyGenesisReward                    uint64
	preDeflationaryPhaseBaseSubsidy         uint64
	coinbasePayloadScriptPublicKeyMaxLength uint8
	genesisHash                             *externalapi.DomainHash
	deflationaryPhaseDaaScore               uint64
	deflationaryPhaseBaseSubsidy            uint64

	databaseContext     model.DBReader
	dagTraversalManager model.DAGTraversalManager
	ghostdagDataStore   model.GHOSTDAGDataStore
	acceptanceDataStore model.AcceptanceDataStore
	daaBlocksStore      model.DAABlocksStore
	blockStore          model.BlockStore
	pruningStore        model.PruningStore
	blockHeaderStore    model.BlockHeaderStore
}

func (c *coinbaseManager) ExpectedCoinbaseTransaction(stagingArea *model.StagingArea, blockHash *externalapi.DomainHash,
	coinbaseData *externalapi.DomainCoinbaseData) (expectedTransaction *externalapi.DomainTransaction, hasRedReward bool, err error) {

	ghostdagData, err := c.ghostdagDataStore.Get(c.databaseContext, stagingArea, blockHash, true)
	if !database.IsNotFoundError(err) && err != nil {
		return nil, false, err
	}

	// If there's ghostdag data with trusted data we prefer it because we need the original merge set non-pruned merge set.
	if database.IsNotFoundError(err) {
		ghostdagData, err = c.ghostdagDataStore.Get(c.databaseContext, stagingArea, blockHash, false)
		if err != nil {
			return nil, false, err
		}
	}

	acceptanceData, err := c.acceptanceDataStore.Get(c.databaseContext, stagingArea, blockHash)
	if err != nil {
		return nil, false, err
	}

	daaAddedBlocksSet, err := c.daaAddedBlocksSet(stagingArea, blockHash)
	if err != nil {
		return nil, false, err
	}

	txOuts := make([]*externalapi.DomainTransactionOutput, 0, len(ghostdagData.MergeSetBlues()))
	acceptanceDataMap := acceptanceDataFromArrayToMap(acceptanceData)
	for _, blue := range ghostdagData.MergeSetBlues() {
		txOut, hasReward, err := c.coinbaseOutputForBlueBlock(stagingArea, blue, acceptanceDataMap[*blue], daaAddedBlocksSet)
		if err != nil {
			return nil, false, err
		}

		if hasReward {
			txOuts = append(txOuts, txOut)
		}
	}

	txOut, hasRedReward, err := c.coinbaseOutputForRewardFromRedBlocks(
		stagingArea, ghostdagData, acceptanceData, daaAddedBlocksSet, coinbaseData)
	if err != nil {
		return nil, false, err
	}

	if hasRedReward {
		txOuts = append(txOuts, txOut)
	}

	subsidy, err := c.CalcBlockSubsidy(stagingArea, blockHash)
	if err != nil {
		return nil, false, err
	}

	payload, err := c.serializeCoinbasePayload(ghostdagData.BlueScore(), coinbaseData, subsidy)
	if err != nil {
		return nil, false, err
	}

	return &externalapi.DomainTransaction{
		Version:      constants.MaxTransactionVersion,
		Inputs:       []*externalapi.DomainTransactionInput{},
		Outputs:      txOuts,
		LockTime:     0,
		SubnetworkID: subnetworks.SubnetworkIDCoinbase,
		Gas:          0,
		Payload:      payload,
	}, hasRedReward, nil
}

func (c *coinbaseManager) daaAddedBlocksSet(stagingArea *model.StagingArea, blockHash *externalapi.DomainHash) (
	hashset.HashSet, error) {

	daaAddedBlocks, err := c.daaBlocksStore.DAAAddedBlocks(c.databaseContext, stagingArea, blockHash)
	if err != nil {
		return nil, err
	}

	return hashset.NewFromSlice(daaAddedBlocks...), nil
}

// coinbaseOutputForBlueBlock calculates the output that should go into the coinbase transaction of blueBlock
// If blueBlock gets no fee - returns nil for txOut
func (c *coinbaseManager) coinbaseOutputForBlueBlock(stagingArea *model.StagingArea,
	blueBlock *externalapi.DomainHash, blockAcceptanceData *externalapi.BlockAcceptanceData,
	mergingBlockDAAAddedBlocksSet hashset.HashSet) (*externalapi.DomainTransactionOutput, bool, error) {

	blockReward, err := c.calcMergedBlockReward(stagingArea, blueBlock, blockAcceptanceData, mergingBlockDAAAddedBlocksSet)
	if err != nil {
		return nil, false, err
	}

	if blockReward == 0 {
		return nil, false, nil
	}

	// the ScriptPublicKey for the coinbase is parsed from the coinbase payload
	_, coinbaseData, _, err := c.ExtractCoinbaseDataBlueScoreAndSubsidy(blockAcceptanceData.TransactionAcceptanceData[0].Transaction)
	if err != nil {
		return nil, false, err
	}

	txOut := &externalapi.DomainTransactionOutput{
		Value:           blockReward,
		ScriptPublicKey: coinbaseData.ScriptPublicKey,
	}

	return txOut, true, nil
}

func (c *coinbaseManager) coinbaseOutputForRewardFromRedBlocks(stagingArea *model.StagingArea,
	ghostdagData *externalapi.BlockGHOSTDAGData, acceptanceData externalapi.AcceptanceData, daaAddedBlocksSet hashset.HashSet,
	coinbaseData *externalapi.DomainCoinbaseData) (*externalapi.DomainTransactionOutput, bool, error) {

	acceptanceDataMap := acceptanceDataFromArrayToMap(acceptanceData)
	totalReward := uint64(0)
	for _, red := range ghostdagData.MergeSetReds() {
		reward, err := c.calcMergedBlockReward(stagingArea, red, acceptanceDataMap[*red], daaAddedBlocksSet)
		if err != nil {
			return nil, false, err
		}

		totalReward += reward
	}

	if totalReward == 0 {
		return nil, false, nil
	}

	return &externalapi.DomainTransactionOutput{
		Value:           totalReward,
		ScriptPublicKey: coinbaseData.ScriptPublicKey,
	}, true, nil
}

func acceptanceDataFromArrayToMap(acceptanceData externalapi.AcceptanceData) map[externalapi.DomainHash]*externalapi.BlockAcceptanceData {
	acceptanceDataMap := make(map[externalapi.DomainHash]*externalapi.BlockAcceptanceData, len(acceptanceData))
	for _, blockAcceptanceData := range acceptanceData {
		acceptanceDataMap[*blockAcceptanceData.BlockHash] = blockAcceptanceData
	}
	return acceptanceDataMap
}

// CalcBlockSubsidy returns the subsidy amount a block at the provided blue score
// should have. This is mainly used for determining how much the coinbase for
// newly generated blocks awards as well as validating the coinbase for blocks
// has the expected value.
func (c *coinbaseManager) CalcBlockSubsidy(stagingArea *model.StagingArea, blockHash *externalapi.DomainHash) (uint64, error) {
	if blockHash.Equal(c.genesisHash) {
		return c.subsidyGenesisReward, nil
	}
	blockDaaScore, err := c.daaBlocksStore.DAAScore(c.databaseContext, stagingArea, blockHash)
	if err != nil {
		return 0, err
	}
	if blockDaaScore < c.deflationaryPhaseDaaScore {
		return c.preDeflationaryPhaseBaseSubsidy, nil
	}

	blockSubsidy := c.calcDeflationaryPeriodBlockSubsidy(blockDaaScore)
	return blockSubsidy, nil
}

func (c *coinbaseManager) calcDeflationaryPeriodBlockSubsidy(blockDaaScore uint64) uint64 {
	// We define a year as 365.25 days and a month as 365.25 / 12 = 30.4375
	// secondsPerMonth = 30.4375 * 24 * 60 * 60
	const secondsPerMonth = 2629800
	// Note that this calculation implicitly assumes that block per second = 1 (by assuming daa score diff is in second units).
	monthsSinceDeflationaryPhaseStarted := (blockDaaScore - c.deflationaryPhaseDaaScore) / secondsPerMonth
	// Return the pre-calculated value from subsidy-per-month table
	return c.getDeflationaryPeriodBlockSubsidyFromTable(monthsSinceDeflationaryPhaseStarted)
}

/*
This table was pre-calculated by calling `calcDeflationaryPeriodBlockSubsidyFloatCalc` for all months until reaching 0 subsidy.
To regenerate this table, run `TestBuildSubsidyTable` in coinbasemanager_test.go (note the `deflationaryPhaseBaseSubsidy` therein)
*/
var subsidyByDeflationaryMonthTable = []uint64{200000000000, 188774862536, 178179743628, 168179283050, 158740105196, 149830707687, 141421356237, 133483985417, 125992104989, 118920711500, 112246204830, 105946309435, 100000000000, 94387431268, 89089871814, 84089641525, 79370052598, 74915353843, 70710678118, 66741992708, 62996052494, 59460355750, 56123102415, 52973154717, 50000000000,
	47193715634, 44544935907, 42044820762, 39685026299, 37457676921, 35355339059, 33370996354, 31498026247, 29730177875, 28061551207, 26486577358, 25000000000, 23596857817, 22272467953, 21022410381, 19842513149, 18728838460, 17677669529, 16685498177, 15749013123, 14865088937, 14030775603, 13243288679, 12500000000, 11798428908,
	11136233976, 10511205190, 9921256574, 9364419230, 8838834764, 8342749088, 7874506561, 7432544468, 7015387801, 6621644339, 6250000000, 5899214454, 5568116988, 5255602595, 4960628287, 4682209615, 4419417382, 4171374544, 3937253280, 3716272234, 3507693900, 3310822169, 3125000000, 2949607227, 2784058494,
	2627801297, 2480314143, 2341104807, 2209708691, 2085687272, 1968626640, 1858136117, 1753846950, 1655411084, 1562500000, 1474803613, 1392029247, 1313900648, 1240157071, 1170552403, 1104854345, 1042843636, 984313320, 929068058, 876923475, 827705542, 781250000, 737401806, 696014623, 656950324,
	620078535, 585276201, 552427172, 521421818, 492156660, 464534029, 438461737, 413852771, 390625000, 368700903, 348007311, 328475162, 310039267, 292638100, 276213586, 260710909, 246078330, 232267014, 219230868, 206926385, 195312500, 184350451, 174003655, 164237581, 155019633,
	146319050, 138106793, 130355454, 123039165, 116133507, 109615434, 103463192, 97656250, 92175225, 87001827, 82118790, 77509816, 73159525, 69053396, 65177727, 61519582, 58066753, 54807717, 51731596, 48828125, 46087612, 43500913, 41059395, 38754908, 36579762,
	34526698, 32588863, 30759791, 29033376, 27403858, 25865798, 24414062, 23043806, 21750456, 20529697, 19377454, 18289881, 17263349, 16294431, 15379895, 14516688, 13701929, 12932899, 12207031, 11521903, 10875228, 10264848, 9688727, 9144940, 8631674,
	8147215, 7689947, 7258344, 6850964, 6466449, 6103515, 5760951, 5437614, 5132424, 4844363, 4572470, 4315837, 4073607, 3844973, 3629172, 3425482, 3233224, 3051757, 2880475, 2718807, 2566212, 2422181, 2286235, 2157918, 2036803,
	1922486, 1814586, 1712741, 1616612, 1525878, 1440237, 1359403, 1283106, 1211090, 1143117, 1078959, 1018401, 961243, 907293, 856370, 808306, 762939, 720118, 679701, 641553, 605545, 571558, 539479, 509200, 480621,
	453646, 428185, 404153, 381469, 360059, 339850, 320776, 302772, 285779, 269739, 254600, 240310, 226823, 214092, 202076, 190734, 180029, 169925, 160388, 151386, 142889, 134869, 127300, 120155, 113411,
	107046, 101038, 95367, 90014, 84962, 80194, 75693, 71444, 67434, 63650, 60077, 56705, 53523, 50519, 47683, 45007, 42481, 40097, 37846, 35722, 33717, 31825, 30038, 28352, 26761,
	25259, 23841, 22503, 21240, 20048, 18923, 17861, 16858, 15912, 15019, 14176, 13380, 12629, 11920, 11251, 10620, 10024, 9461, 8930, 8429, 7956, 7509, 7088, 6690, 6314,
	5960, 5625, 5310, 5012, 4730, 4465, 4214, 3978, 3754, 3544, 3345, 3157, 2980, 2812, 2655, 2506, 2365, 2232, 2107, 1989, 1877, 1772, 1672, 1578, 1490,
	1406, 1327, 1253, 1182, 1116, 1053, 994, 938, 886, 836, 789, 745, 703, 663, 626, 591, 558, 526, 497, 469, 443, 418, 394, 372, 351,
	331, 313, 295, 279, 263, 248, 234, 221, 209, 197, 186, 175, 165, 156, 147, 139, 131, 124, 117, 110, 104, 98, 93, 87, 82,
	78, 73, 69, 65, 62, 58, 55, 52, 49, 46, 43, 41, 39, 36, 34, 32, 31, 29, 27, 26, 24, 23, 21, 20, 19,
	18, 17, 16, 15, 14, 13, 13, 12, 11, 10, 10, 9, 9, 8, 8, 7, 7, 6, 6, 6, 5, 5, 5, 4, 4,
	4, 4, 3, 3, 3, 3, 3, 2, 2, 2, 2, 2, 2, 2, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1,
	1, 0,
}

func (c *coinbaseManager) getDeflationaryPeriodBlockSubsidyFromTable(month uint64) uint64 {
	if month >= uint64(len(subsidyByDeflationaryMonthTable)) {
		month = uint64(len(subsidyByDeflationaryMonthTable) - 1)
	}
	return subsidyByDeflationaryMonthTable[month]
}

func (c *coinbaseManager) calcDeflationaryPeriodBlockSubsidyFloatCalc(month uint64) uint64 {
	baseSubsidy := c.deflationaryPhaseBaseSubsidy
	subsidy := float64(baseSubsidy) / math.Pow(2, float64(month)/12)
	return uint64(subsidy)
}

func (c *coinbaseManager) calcMergedBlockReward(stagingArea *model.StagingArea, blockHash *externalapi.DomainHash,
	blockAcceptanceData *externalapi.BlockAcceptanceData, mergingBlockDAAAddedBlocksSet hashset.HashSet) (uint64, error) {

	if !blockHash.Equal(blockAcceptanceData.BlockHash) {
		return 0, errors.Errorf("blockAcceptanceData.BlockHash is expected to be %s but got %s",
			blockHash, blockAcceptanceData.BlockHash)
	}

	if !mergingBlockDAAAddedBlocksSet.Contains(blockHash) {
		return 0, nil
	}

	totalFees := uint64(0)
	for _, txAcceptanceData := range blockAcceptanceData.TransactionAcceptanceData {
		if txAcceptanceData.IsAccepted {
			totalFees += txAcceptanceData.Fee
		}
	}

	block, err := c.blockStore.Block(c.databaseContext, stagingArea, blockHash)
	if err != nil {
		return 0, err
	}

	_, _, subsidy, err := c.ExtractCoinbaseDataBlueScoreAndSubsidy(block.Transactions[transactionhelper.CoinbaseTransactionIndex])
	if err != nil {
		return 0, err
	}

	return subsidy + totalFees, nil
}

// New instantiates a new CoinbaseManager
func New(
	databaseContext model.DBReader,

	subsidyGenesisReward uint64,
	preDeflationaryPhaseBaseSubsidy uint64,
	coinbasePayloadScriptPublicKeyMaxLength uint8,
	genesisHash *externalapi.DomainHash,
	deflationaryPhaseDaaScore uint64,
	deflationaryPhaseBaseSubsidy uint64,

	dagTraversalManager model.DAGTraversalManager,
	ghostdagDataStore model.GHOSTDAGDataStore,
	acceptanceDataStore model.AcceptanceDataStore,
	daaBlocksStore model.DAABlocksStore,
	blockStore model.BlockStore,
	pruningStore model.PruningStore,
	blockHeaderStore model.BlockHeaderStore) model.CoinbaseManager {

	return &coinbaseManager{
		databaseContext: databaseContext,

		subsidyGenesisReward:                    subsidyGenesisReward,
		preDeflationaryPhaseBaseSubsidy:         preDeflationaryPhaseBaseSubsidy,
		coinbasePayloadScriptPublicKeyMaxLength: coinbasePayloadScriptPublicKeyMaxLength,
		genesisHash:                             genesisHash,
		deflationaryPhaseDaaScore:               deflationaryPhaseDaaScore,
		deflationaryPhaseBaseSubsidy:            deflationaryPhaseBaseSubsidy,

		dagTraversalManager: dagTraversalManager,
		ghostdagDataStore:   ghostdagDataStore,
		acceptanceDataStore: acceptanceDataStore,
		daaBlocksStore:      daaBlocksStore,
		blockStore:          blockStore,
		pruningStore:        pruningStore,
		blockHeaderStore:    blockHeaderStore,
	}
}
