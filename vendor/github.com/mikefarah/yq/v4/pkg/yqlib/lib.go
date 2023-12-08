// Use the top level Evaluator or StreamEvaluator to evaluate expressions and return matches.
package yqlib

import (
	"container/list"
	"fmt"
	"math"
	"strconv"
	"strings"

	logging "gopkg.in/op/go-logging.v1"
)

var ExpressionParser ExpressionParserInterface

func InitExpressionParser() {
	if ExpressionParser == nil {
		ExpressionParser = newExpressionParser()
	}
}

var log = logging.MustGetLogger("yq-lib")

var PrettyPrintExp = `(... | (select(tag != "!!str"), select(tag == "!!str") | select(test("(?i)^(y|yes|n|no|on|off)$") | not))  ) style=""`

// GetLogger returns the yq logger instance.
func GetLogger() *logging.Logger {
	return log
}

type operationType struct {
	Type       string
	NumArgs    uint // number of arguments to the op
	Precedence uint
	Handler    operatorHandler
}

var orOpType = &operationType{Type: "OR", NumArgs: 2, Precedence: 20, Handler: orOperator}
var andOpType = &operationType{Type: "AND", NumArgs: 2, Precedence: 20, Handler: andOperator}
var reduceOpType = &operationType{Type: "REDUCE", NumArgs: 2, Precedence: 35, Handler: reduceOperator}

var blockOpType = &operationType{Type: "BLOCK", Precedence: 10, NumArgs: 2, Handler: emptyOperator}

var unionOpType = &operationType{Type: "UNION", NumArgs: 2, Precedence: 10, Handler: unionOperator}

var pipeOpType = &operationType{Type: "PIPE", NumArgs: 2, Precedence: 30, Handler: pipeOperator}

var assignOpType = &operationType{Type: "ASSIGN", NumArgs: 2, Precedence: 40, Handler: assignUpdateOperator}
var addAssignOpType = &operationType{Type: "ADD_ASSIGN", NumArgs: 2, Precedence: 40, Handler: addAssignOperator}
var subtractAssignOpType = &operationType{Type: "SUBTRACT_ASSIGN", NumArgs: 2, Precedence: 40, Handler: subtractAssignOperator}

var assignAttributesOpType = &operationType{Type: "ASSIGN_ATTRIBUTES", NumArgs: 2, Precedence: 40, Handler: assignAttributesOperator}
var assignStyleOpType = &operationType{Type: "ASSIGN_STYLE", NumArgs: 2, Precedence: 40, Handler: assignStyleOperator}
var assignVariableOpType = &operationType{Type: "ASSIGN_VARIABLE", NumArgs: 2, Precedence: 40, Handler: useWithPipe}
var assignTagOpType = &operationType{Type: "ASSIGN_TAG", NumArgs: 2, Precedence: 40, Handler: assignTagOperator}
var assignCommentOpType = &operationType{Type: "ASSIGN_COMMENT", NumArgs: 2, Precedence: 40, Handler: assignCommentsOperator}
var assignAnchorOpType = &operationType{Type: "ASSIGN_ANCHOR", NumArgs: 2, Precedence: 40, Handler: assignAnchorOperator}
var assignAliasOpType = &operationType{Type: "ASSIGN_ALIAS", NumArgs: 2, Precedence: 40, Handler: assignAliasOperator}

var multiplyOpType = &operationType{Type: "MULTIPLY", NumArgs: 2, Precedence: 42, Handler: multiplyOperator}
var multiplyAssignOpType = &operationType{Type: "MULTIPLY_ASSIGN", NumArgs: 2, Precedence: 42, Handler: multiplyAssignOperator}

var divideOpType = &operationType{Type: "DIVIDE", NumArgs: 2, Precedence: 42, Handler: divideOperator}

var moduloOpType = &operationType{Type: "MODULO", NumArgs: 2, Precedence: 42, Handler: moduloOperator}

var addOpType = &operationType{Type: "ADD", NumArgs: 2, Precedence: 42, Handler: addOperator}
var subtractOpType = &operationType{Type: "SUBTRACT", NumArgs: 2, Precedence: 42, Handler: subtractOperator}
var alternativeOpType = &operationType{Type: "ALTERNATIVE", NumArgs: 2, Precedence: 42, Handler: alternativeOperator}

var equalsOpType = &operationType{Type: "EQUALS", NumArgs: 2, Precedence: 40, Handler: equalsOperator}
var notEqualsOpType = &operationType{Type: "NOT_EQUALS", NumArgs: 2, Precedence: 40, Handler: notEqualsOperator}

var compareOpType = &operationType{Type: "COMPARE", NumArgs: 2, Precedence: 40, Handler: compareOperator}

// createmap needs to be above union, as we use union to build the components of the objects
var createMapOpType = &operationType{Type: "CREATE_MAP", NumArgs: 2, Precedence: 15, Handler: createMapOperator}

var shortPipeOpType = &operationType{Type: "SHORT_PIPE", NumArgs: 2, Precedence: 45, Handler: pipeOperator}

var lengthOpType = &operationType{Type: "LENGTH", NumArgs: 0, Precedence: 50, Handler: lengthOperator}
var lineOpType = &operationType{Type: "LINE", NumArgs: 0, Precedence: 50, Handler: lineOperator}
var columnOpType = &operationType{Type: "LINE", NumArgs: 0, Precedence: 50, Handler: columnOperator}

var expressionOpType = &operationType{Type: "EXP", NumArgs: 0, Precedence: 50, Handler: expressionOperator}

var collectOpType = &operationType{Type: "COLLECT", NumArgs: 1, Precedence: 50, Handler: collectOperator}
var mapOpType = &operationType{Type: "MAP", NumArgs: 1, Precedence: 50, Handler: mapOperator}
var filterOpType = &operationType{Type: "FILTER", NumArgs: 1, Precedence: 50, Handler: filterOperator}
var errorOpType = &operationType{Type: "ERROR", NumArgs: 1, Precedence: 50, Handler: errorOperator}
var pickOpType = &operationType{Type: "PICK", NumArgs: 1, Precedence: 50, Handler: pickOperator}
var evalOpType = &operationType{Type: "EVAL", NumArgs: 1, Precedence: 50, Handler: evalOperator}
var mapValuesOpType = &operationType{Type: "MAP_VALUES", NumArgs: 1, Precedence: 50, Handler: mapValuesOperator}

var formatDateTimeOpType = &operationType{Type: "FORMAT_DATE_TIME", NumArgs: 1, Precedence: 50, Handler: formatDateTime}
var withDtFormatOpType = &operationType{Type: "WITH_DATE_TIME_FORMAT", NumArgs: 1, Precedence: 50, Handler: withDateTimeFormat}
var nowOpType = &operationType{Type: "NOW", NumArgs: 0, Precedence: 50, Handler: nowOp}
var tzOpType = &operationType{Type: "TIMEZONE", NumArgs: 1, Precedence: 50, Handler: tzOp}
var fromUnixOpType = &operationType{Type: "FROM_UNIX", NumArgs: 0, Precedence: 50, Handler: fromUnixOp}
var toUnixOpType = &operationType{Type: "TO_UNIX", NumArgs: 0, Precedence: 50, Handler: toUnixOp}

var encodeOpType = &operationType{Type: "ENCODE", NumArgs: 0, Precedence: 50, Handler: encodeOperator}
var decodeOpType = &operationType{Type: "DECODE", NumArgs: 0, Precedence: 50, Handler: decodeOperator}

var anyOpType = &operationType{Type: "ANY", NumArgs: 0, Precedence: 50, Handler: anyOperator}
var allOpType = &operationType{Type: "ALL", NumArgs: 0, Precedence: 50, Handler: allOperator}
var containsOpType = &operationType{Type: "CONTAINS", NumArgs: 1, Precedence: 50, Handler: containsOperator}
var anyConditionOpType = &operationType{Type: "ANY_CONDITION", NumArgs: 1, Precedence: 50, Handler: anyOperator}
var allConditionOpType = &operationType{Type: "ALL_CONDITION", NumArgs: 1, Precedence: 50, Handler: allOperator}

var toEntriesOpType = &operationType{Type: "TO_ENTRIES", NumArgs: 0, Precedence: 50, Handler: toEntriesOperator}
var fromEntriesOpType = &operationType{Type: "FROM_ENTRIES", NumArgs: 0, Precedence: 50, Handler: fromEntriesOperator}
var withEntriesOpType = &operationType{Type: "WITH_ENTRIES", NumArgs: 1, Precedence: 50, Handler: withEntriesOperator}

var withOpType = &operationType{Type: "WITH", NumArgs: 1, Precedence: 50, Handler: withOperator}

var splitDocumentOpType = &operationType{Type: "SPLIT_DOC", NumArgs: 0, Precedence: 50, Handler: splitDocumentOperator}
var getVariableOpType = &operationType{Type: "GET_VARIABLE", NumArgs: 0, Precedence: 55, Handler: getVariableOperator}
var getStyleOpType = &operationType{Type: "GET_STYLE", NumArgs: 0, Precedence: 50, Handler: getStyleOperator}
var getTagOpType = &operationType{Type: "GET_TAG", NumArgs: 0, Precedence: 50, Handler: getTagOperator}
var getKindOpType = &operationType{Type: "GET_KIND", NumArgs: 0, Precedence: 50, Handler: getKindOperator}

var getKeyOpType = &operationType{Type: "GET_KEY", NumArgs: 0, Precedence: 50, Handler: getKeyOperator}
var isKeyOpType = &operationType{Type: "IS_KEY", NumArgs: 0, Precedence: 50, Handler: isKeyOperator}
var getParentOpType = &operationType{Type: "GET_PARENT", NumArgs: 0, Precedence: 50, Handler: getParentOperator}

var getCommentOpType = &operationType{Type: "GET_COMMENT", NumArgs: 0, Precedence: 50, Handler: getCommentsOperator}
var getAnchorOpType = &operationType{Type: "GET_ANCHOR", NumArgs: 0, Precedence: 50, Handler: getAnchorOperator}
var getAliasOpType = &operationType{Type: "GET_ALIAS", NumArgs: 0, Precedence: 50, Handler: getAliasOperator}
var getDocumentIndexOpType = &operationType{Type: "GET_DOCUMENT_INDEX", NumArgs: 0, Precedence: 50, Handler: getDocumentIndexOperator}
var getFilenameOpType = &operationType{Type: "GET_FILENAME", NumArgs: 0, Precedence: 50, Handler: getFilenameOperator}
var getFileIndexOpType = &operationType{Type: "GET_FILE_INDEX", NumArgs: 0, Precedence: 50, Handler: getFileIndexOperator}

var getPathOpType = &operationType{Type: "GET_PATH", NumArgs: 0, Precedence: 50, Handler: getPathOperator}
var setPathOpType = &operationType{Type: "SET_PATH", NumArgs: 1, Precedence: 50, Handler: setPathOperator}
var delPathsOpType = &operationType{Type: "DEL_PATHS", NumArgs: 1, Precedence: 50, Handler: delPathsOperator}

var explodeOpType = &operationType{Type: "EXPLODE", NumArgs: 1, Precedence: 50, Handler: explodeOperator}
var sortByOpType = &operationType{Type: "SORT_BY", NumArgs: 1, Precedence: 50, Handler: sortByOperator}
var reverseOpType = &operationType{Type: "REVERSE", NumArgs: 0, Precedence: 50, Handler: reverseOperator}
var sortOpType = &operationType{Type: "SORT", NumArgs: 0, Precedence: 50, Handler: sortOperator}
var shuffleOpType = &operationType{Type: "SHUFFLE", NumArgs: 0, Precedence: 50, Handler: shuffleOperator}

var sortKeysOpType = &operationType{Type: "SORT_KEYS", NumArgs: 1, Precedence: 50, Handler: sortKeysOperator}

var joinStringOpType = &operationType{Type: "JOIN", NumArgs: 1, Precedence: 50, Handler: joinStringOperator}
var subStringOpType = &operationType{Type: "SUBSTR", NumArgs: 1, Precedence: 50, Handler: substituteStringOperator}
var matchOpType = &operationType{Type: "MATCH", NumArgs: 1, Precedence: 50, Handler: matchOperator}
var captureOpType = &operationType{Type: "CAPTURE", NumArgs: 1, Precedence: 50, Handler: captureOperator}
var testOpType = &operationType{Type: "TEST", NumArgs: 1, Precedence: 50, Handler: testOperator}
var splitStringOpType = &operationType{Type: "SPLIT", NumArgs: 1, Precedence: 50, Handler: splitStringOperator}
var changeCaseOpType = &operationType{Type: "CHANGE_CASE", NumArgs: 0, Precedence: 50, Handler: changeCaseOperator}
var trimOpType = &operationType{Type: "TRIM", NumArgs: 0, Precedence: 50, Handler: trimSpaceOperator}

var loadOpType = &operationType{Type: "LOAD", NumArgs: 1, Precedence: 50, Handler: loadYamlOperator}

var keysOpType = &operationType{Type: "KEYS", NumArgs: 0, Precedence: 50, Handler: keysOperator}

var collectObjectOpType = &operationType{Type: "COLLECT_OBJECT", NumArgs: 0, Precedence: 50, Handler: collectObjectOperator}
var traversePathOpType = &operationType{Type: "TRAVERSE_PATH", NumArgs: 0, Precedence: 55, Handler: traversePathOperator}
var traverseArrayOpType = &operationType{Type: "TRAVERSE_ARRAY", NumArgs: 2, Precedence: 50, Handler: traverseArrayOperator}

var selfReferenceOpType = &operationType{Type: "SELF", NumArgs: 0, Precedence: 55, Handler: selfOperator}
var valueOpType = &operationType{Type: "VALUE", NumArgs: 0, Precedence: 50, Handler: valueOperator}
var referenceOpType = &operationType{Type: "REF", NumArgs: 0, Precedence: 50, Handler: referenceOperator}
var envOpType = &operationType{Type: "ENV", NumArgs: 0, Precedence: 50, Handler: envOperator}
var notOpType = &operationType{Type: "NOT", NumArgs: 0, Precedence: 50, Handler: notOperator}
var toNumberOpType = &operationType{Type: "TO_NUMBER", NumArgs: 0, Precedence: 50, Handler: toNumberOperator}
var emptyOpType = &operationType{Type: "EMPTY", Precedence: 50, Handler: emptyOperator}

var envsubstOpType = &operationType{Type: "ENVSUBST", NumArgs: 0, Precedence: 50, Handler: envsubstOperator}

var recursiveDescentOpType = &operationType{Type: "RECURSIVE_DESCENT", NumArgs: 0, Precedence: 50, Handler: recursiveDescentOperator}

var selectOpType = &operationType{Type: "SELECT", NumArgs: 1, Precedence: 50, Handler: selectOperator}
var hasOpType = &operationType{Type: "HAS", NumArgs: 1, Precedence: 50, Handler: hasOperator}
var uniqueOpType = &operationType{Type: "UNIQUE", NumArgs: 0, Precedence: 50, Handler: unique}
var uniqueByOpType = &operationType{Type: "UNIQUE_BY", NumArgs: 1, Precedence: 50, Handler: uniqueBy}
var groupByOpType = &operationType{Type: "GROUP_BY", NumArgs: 1, Precedence: 50, Handler: groupBy}
var flattenOpType = &operationType{Type: "FLATTEN_BY", NumArgs: 0, Precedence: 50, Handler: flattenOp}
var deleteChildOpType = &operationType{Type: "DELETE", NumArgs: 1, Precedence: 40, Handler: deleteChildOperator}

type Operation struct {
	OperationType *operationType
	Value         interface{}
	StringValue   string
	CandidateNode *CandidateNode // used for Value Path elements
	Preferences   interface{}
	UpdateAssign  bool // used for assign ops, when true it means we evaluate the rhs given the lhs
}

func recurseNodeArrayEqual(lhs *CandidateNode, rhs *CandidateNode) bool {
	if len(lhs.Content) != len(rhs.Content) {
		return false
	}

	for index := 0; index < len(lhs.Content); index = index + 1 {
		if !recursiveNodeEqual(lhs.Content[index], rhs.Content[index]) {
			return false
		}
	}
	return true
}

func findInArray(array *CandidateNode, item *CandidateNode) int {

	for index := 0; index < len(array.Content); index = index + 1 {
		if recursiveNodeEqual(array.Content[index], item) {
			return index
		}
	}
	return -1
}

func findKeyInMap(dataMap *CandidateNode, item *CandidateNode) int {

	for index := 0; index < len(dataMap.Content); index = index + 2 {
		if recursiveNodeEqual(dataMap.Content[index], item) {
			return index
		}
	}
	return -1
}

func recurseNodeObjectEqual(lhs *CandidateNode, rhs *CandidateNode) bool {
	if len(lhs.Content) != len(rhs.Content) {
		return false
	}

	for index := 0; index < len(lhs.Content); index = index + 2 {
		key := lhs.Content[index]
		value := lhs.Content[index+1]

		indexInRHS := findInArray(rhs, key)

		if indexInRHS == -1 || !recursiveNodeEqual(value, rhs.Content[indexInRHS+1]) {
			return false
		}
	}
	return true
}

func parseSnippet(value string) (*CandidateNode, error) {
	if value == "" {
		return &CandidateNode{
			Kind: ScalarNode,
			Tag:  "!!null",
		}, nil
	}
	decoder := NewYamlDecoder(ConfiguredYamlPreferences)
	err := decoder.Init(strings.NewReader(value))
	if err != nil {
		return nil, err
	}
	result, err := decoder.Decode()
	if err != nil {
		return nil, err
	}
	result.Line = 0
	result.Column = 0
	return result, err
}

func recursiveNodeEqual(lhs *CandidateNode, rhs *CandidateNode) bool {
	if lhs.Kind != rhs.Kind {
		return false
	}

	if lhs.Kind == ScalarNode {
		//process custom tags of scalar nodes.
		//dont worry about matching tags of maps or arrays.

		lhsTag := lhs.guessTagFromCustomType()
		rhsTag := rhs.guessTagFromCustomType()

		if lhsTag != rhsTag {
			return false
		}
	}

	if lhs.Tag == "!!null" {
		return true

	} else if lhs.Kind == ScalarNode {
		return lhs.Value == rhs.Value
	} else if lhs.Kind == SequenceNode {
		return recurseNodeArrayEqual(lhs, rhs)
	} else if lhs.Kind == MappingNode {
		return recurseNodeObjectEqual(lhs, rhs)
	}
	return false
}

// yaml numbers can be hex and octal encoded...
func parseInt64(numberString string) (string, int64, error) {
	if strings.HasPrefix(numberString, "0x") ||
		strings.HasPrefix(numberString, "0X") {
		num, err := strconv.ParseInt(numberString[2:], 16, 64)
		return "0x%X", num, err
	} else if strings.HasPrefix(numberString, "0o") {
		num, err := strconv.ParseInt(numberString[2:], 8, 64)
		return "0o%o", num, err
	}
	num, err := strconv.ParseInt(numberString, 10, 64)
	return "%v", num, err
}

func parseInt(numberString string) (int, error) {
	_, parsed, err := parseInt64(numberString)

	if err != nil {
		return 0, err
	} else if parsed > math.MaxInt || parsed < math.MinInt {
		return 0, fmt.Errorf("%v is not within [%v, %v]", parsed, math.MinInt, math.MaxInt)
	}

	return int(parsed), err
}

func headAndLineComment(node *CandidateNode) string {
	return headComment(node) + lineComment(node)
}

func headComment(node *CandidateNode) string {
	return strings.Replace(node.HeadComment, "#", "", 1)
}

func lineComment(node *CandidateNode) string {
	return strings.Replace(node.LineComment, "#", "", 1)
}

func footComment(node *CandidateNode) string {
	return strings.Replace(node.FootComment, "#", "", 1)
}

func createValueOperation(value interface{}, stringValue string) *Operation {
	log.Debug("creating value op for string %v", stringValue)
	var node = createScalarNode(value, stringValue)

	return &Operation{
		OperationType: valueOpType,
		Value:         value,
		StringValue:   stringValue,
		CandidateNode: node,
	}
}

// debugging purposes only
func (p *Operation) toString() string {
	if p == nil {
		return "OP IS NIL"
	}
	if p.OperationType == traversePathOpType {
		return fmt.Sprintf("%v", p.Value)
	} else if p.OperationType == selfReferenceOpType {
		return "SELF"
	} else if p.OperationType == valueOpType {
		return fmt.Sprintf("%v (%T)", p.Value, p.Value)
	} else {
		return fmt.Sprintf("%v", p.OperationType.Type)
	}
}

// use for debugging only
func NodesToString(collection *list.List) string {
	if !log.IsEnabledFor(logging.DEBUG) {
		return ""
	}

	result := fmt.Sprintf("%v results\n", collection.Len())
	for el := collection.Front(); el != nil; el = el.Next() {
		result = result + "\n" + NodeToString(el.Value.(*CandidateNode))
	}
	return result
}

func NodeToString(node *CandidateNode) string {
	if !log.IsEnabledFor(logging.DEBUG) {
		return ""
	}
	if node == nil {
		return "-- nil --"
	}
	tag := node.Tag
	if node.Kind == AliasNode {
		tag = "alias"
	}
	valueToUse := node.Value
	if valueToUse == "" {
		valueToUse = fmt.Sprintf("%v kids", len(node.Content))
	}
	return fmt.Sprintf(`D%v, P%v, %v (%v)::%v`, node.GetDocument(), node.GetNicePath(), KindString(node.Kind), tag, valueToUse)
}

func NodeContentToString(node *CandidateNode, depth int) string {
	if !log.IsEnabledFor(logging.DEBUG) {
		return ""
	}
	var sb strings.Builder
	for _, child := range node.Content {
		for i := 0; i < depth; i++ {
			sb.WriteString(" ")
		}
		sb.WriteString("- ")
		sb.WriteString(NodeToString(child))
		sb.WriteString("\n")
		sb.WriteString(NodeContentToString(child, depth+1))
	}
	return sb.String()
}

func KindString(kind Kind) string {
	switch kind {
	case ScalarNode:
		return "ScalarNode"
	case SequenceNode:
		return "SequenceNode"
	case MappingNode:
		return "MappingNode"
	case AliasNode:
		return "AliasNode"
	default:
		return "unknown!"
	}
}
