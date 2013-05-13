package config

import (
	"bufio"
	"errors"
	"io"
	"os"
	"strings"
)

var (
	commentString   = []string{"#", ";"}
	separatorString = []string{"="}
	validFile       = []string{"ini", "conf"}
)

type Node struct {
	Name  string
	Value string
}

type Section struct {
	Name  string
	Nodes map[string]Node
}

type IniFile struct {
	file     *os.File
	pattern  *Pattern
	Sections map[string]Section
}

type Identifer struct {
	Begin string
	End   string
}

type Pattern struct {
	comments         []string
	separators       []string
	charEnter        string
	bContentRawRead  bool
	sectionIdentifer Identifer
}

func (inifile *IniFile) Parse() error {
	defer inifile.file.Close()
	p := inifile.pattern
	var currSection string
	buff := bufio.NewReader(inifile.file)
	for {
		//parse the cotent line by line
		line, err := buff.ReadString('\n')
		if err == io.EOF {
			break
		} else if err != nil {
			return err
		}
		line = strings.TrimSpace(line)

		//if this line is a blank line or commented,then ignore it
		if isBlankLine(line) || isComments(line, p.comments) {
			continue
		}

		//read section 
		secName := inifile.ReadSection(line)
		if len(secName) != 0 {
			currSection = secName
		}

		//read node
		inifile.ReadNode(currSection, line)
	}
	return nil
}
func (inifile *IniFile) SetPattern(patt *Pattern) bool {
	inifile.pattern = patt
	return true
}

func (section *Section) GetNode(name string) *Node {
	if node, ok := section.Nodes[name]; ok {
		return &node
	}
	return nil
}

func (inifile *IniFile) GetSection(sectionName string) *Section {
	if section, ok := inifile.Sections[sectionName]; ok {
		return &section
	}
	return nil
}

func (inifile *IniFile) ReadNode(secName, line string) bool {
	if len(secName) == 0 {
		return false
	}
	// fmt.Println("section ", secName, "readed")
	// fmt.Println(line)
	index := strings.IndexAny(line, "=")
	// fmt.Println(index)
	if index > 0 {
		//node and value
		nodeName := line[0:index]
		value := line[index+1:]

		//if do not read the raw strings with spacees in the left and right
		//then trim them
		if inifile.pattern.bContentRawRead == false {
			nodeName = strings.TrimSpace(nodeName)
			value = strings.TrimSpace(value)
		}

		inifile.Sections[secName].Nodes[nodeName] = Node{Name: nodeName, Value: value}
		return true
	}
	return false
}

func (inifile *IniFile) ReadSection(line string) string {
	indexBegin := strings.Index(line, inifile.pattern.sectionIdentifer.Begin)
	indexEnd := strings.LastIndex(line, inifile.pattern.sectionIdentifer.End)
	if indexBegin == 0 && indexEnd == len(line)-len(inifile.pattern.sectionIdentifer.End) {
		indexBegin = len(inifile.pattern.sectionIdentifer.Begin)
		secName := line[indexBegin:indexEnd]
		if inifile.AddSection(secName) != nil {
			return secName
		}
	}
	return ""
}

func (inifile *IniFile) AddSection(secName string) *Section {
	if len(secName) == 0 {
		return nil
	}

	sec := &Section{Name: secName, Nodes: make(map[string]Node)}

	inifile.Sections[secName] = *sec
	return sec
}

func isComments(line string, comments []string) bool {
	for _, v := range comments {
		index := strings.Index(line, v)
		if index == 0 {
			return true
		}
	}
	return false
}

func isBlankLine(line string) bool {
	if len(line) == 0 {
		return true
	}
	return false
}

func ValidFile(name string) (*os.File, error) {
	//do not give me files other than .ini files
	index := strings.LastIndex(name, ".")
	exist := false
	for _, v := range validFile {
		if v == name[index+1:] {
			exist = true
		}
	}
	if !exist {
		return nil, errors.New("this is not ini file")
	}
	//load the file from disk 
	file, err := os.Open(name)
	if err != nil {
		return nil, err
	}
	return file, nil
}

func OpenIniFile(name string) (*IniFile, error) {
	// new a inifile obj
	inifile := new(IniFile)

	//init date of the inifile
	f, err := ValidFile(name)
	if err != nil {
		return nil, err
	}
	inifile.file = f
	inifile.Sections = make(map[string]Section)
	//use default pattern to parse the ini file
	inifile.SetPattern(NewPattern())

	return inifile, nil
}

func (inifile *IniFile) String(sectionName, nodeName string) (value string, err error) {
	if sec := inifile.GetSection(sectionName); sec != nil {
		if node := sec.GetNode(nodeName); node != nil {
			return node.Value, nil
		}
		return "", errors.New(nodeName + " does not exist.")
	}
	return "", errors.New(sectionName + " does not exist.")
}

func (inifile *IniFile) Bool() {

}

func NewIniFile() *IniFile {
	ini := new(IniFile)
	ini.pattern = NewPattern()
	ini.Sections = make(map[string]Section)
	return ini
}

func NewPattern() *Pattern {
	patt := Pattern{}
	patt.bContentRawRead = false
	patt.comments = []string{";", "#"}
	if strings.Contains(strings.ToLower(os.Getenv("OS")), "windows") {
		patt.charEnter = "\r\n"
	} else {
		patt.charEnter = "\n"
	}
	patt.separators = []string{"="}
	patt.sectionIdentifer = Identifer{Begin: "[", End: "]"}
	return &patt
}

func (inifile *IniFile) WriteIniFile(fileName, comments string) error {
	file, err := os.OpenFile(fileName, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}

	buf := bufio.NewWriter(file)
	if err := inifile.write(buf, comments); err != nil {
		return err
	}

	buf.Flush()
	return file.Close()
}

func (inifile *IniFile) write(buf *bufio.Writer, comments string) error {
	//write some comments in the first line of the file
	if comments != "" {
		comment := inifile.pattern.comments[0]
		if index := strings.Index(comments, inifile.pattern.charEnter); index != -1 {
			comments = strings.Replace(comments, inifile.pattern.charEnter, inifile.pattern.charEnter+comment, -1)
		}
		if _, err := buf.WriteString(comment + comments + inifile.pattern.charEnter); err != nil {
			return err
		}
	}

	separetor := inifile.pattern.separators[0]
	for _, v := range inifile.Sections {
		if _, err := buf.WriteString(inifile.pattern.charEnter + "[" + v.Name + "]" + inifile.pattern.charEnter); err != nil {
			return err
		}
		for _, value := range v.Nodes {
			if _, err := buf.WriteString(value.Name + separetor + value.Value + inifile.pattern.charEnter); err != nil {
				return err
			}
		}
	}
	return nil
}

func (section *Section) AddNode(name, value string) *Node {
	node := Node{Name: name, Value: value}
	section.Nodes[node.Name] = node
	return &node
}

// func main() {
// 	ini, err := OpenIniFile("1.ini")
// 	if err != nil {
// 		fmt.Println(err.Error())
// 	}

// 	if ini.Parse() == false {
// 		return
// 	}

// 	server, err := ini.String("LangCfg", "path")
// 	if err != nil {
// 		fmt.Println(err.Error())
// 	}
// 	fmt.Println(server)

// 	f := NewIniFile()
// 	sec := f.AddSection("main")
// 	sec.AddNode("server", "10.20.72.213")

// 	f.WriteIniFile("w.ini", "There are music list info")
// }
