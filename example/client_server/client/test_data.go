package main

var test_string = `e32qr213343wg43wg4w3g4wdbsrbsrw3e3$#@#%$%^$)_8998t349t43gw34btrsbtrsh4a122176i87v是然而色不同认识你是32它35654
deafeawveavre热水别人色graebrseverah546I*&#Cvraevae21@%^_+}":?FEWAGAFEAWF3AgsgrseSERresgrevrsevrsevrset4w3tg43vreaoooaaaaaaaaa
ceawvewaveaz##!@R#$$^$%%@#%$@#%$%$%vaewvesrvaervewaraewrawvewavaervae3RQ235*()*(r#@)(*rJLKJKF3QF2Q21	r3ffaf@@R@$#F#@T$@F$#
faef3ag43w@!%#$vrevawre))*(CV:"":v3e3af323vgv@#R$#G$#V$f33f3fccq11cccccccccccccccccccccccccccccccccccccccccccccccccccccccccc
aaaaaaaaaaaaaaaaQQQQQQQQQQQQQQQQCCCCCCCCCCCCCCCCC############F$VSCCCCCCCCCCCCCCCCCCCCCCCCCCAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAACC
cawceawvewa32rf2@#F###################################DDDDDDDDDDDDDDD_________+)(+*^%L;';''CWECweversVEE	afw3feavev
package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path"
	"runtime"
	"strings"

	mysql_base "github.com/huoshan017/mysql-go/base"
	mysql_generate "github.com/huoshan017/mysql-go/generate"
)

//var protoc_root = os.Getenv("GOPATH") + "/mysql-go/_external/"

var protoc_dest_map = map[string]string{
	"windows": "windows/protoc.exe",
	"linux":   "linux/protoc",
	"darwin":  "darwin/protoc",
}

func main() {
	if len(os.Args) < 4 {
		fmt.Fprintf(os.Stderr, "args num not enough\n")
		return
	}

	var arg_config_file, arg_dest_path, arg_protoc_path *string
	// 代碼生成配置路徑
	arg_config_file = flag.String("c", "", "config file path")
	// 目標代碼生成目錄
	arg_dest_path = flag.String("d", "", "dest source path")
	// protoc根目錄
	arg_protoc_path = flag.String("p", "", "protoc file root path")
	flag.Parse()

	var config_path string
	if nil != arg_config_file && *arg_config_file != "" {
		config_path = *arg_config_file
		fmt.Fprintf(os.Stdout, "config file path %v\n", config_path)
	} else {
		fmt.Fprintf(os.Stderr, "not found config file arg\n")
		return
	}

	var dest_path string
	if nil != arg_dest_path && *arg_dest_path != "" {
		dest_path = *arg_dest_path
		fmt.Fprintf(os.Stdout, "dest path %v\n", dest_path)
	} else {
		fmt.Fprintf(os.Stderr, "not found dest path arg\n")
		return
	}

	var protoc_path string
	if nil != arg_protoc_path && *arg_protoc_path != "" {
		go_os := runtime.GOOS //os.Getenv("GOOS")
		protoc_path = *arg_protoc_path + protoc_dest_map[go_os]
	} else {
		fmt.Fprintf(os.Stderr, "not found dest protoc file root path\n")
		return
	}

	fmt.Fprintf(os.Stdout, "protoc path %v\n", protoc_path)

	var config_loader mysql_generate.ConfigLoader
	if !config_loader.Load(config_path) {
		return
	}

	if !config_loader.Generate(dest_path) {
		return
	}

	fmt.Fprintf(os.Stdout, "generated source\n")

	proto_dest_path, config_file := path.Split(config_path)
	proto_dest_path += ".proto/"
	mysql_base.CreateDirs(proto_dest_path)
	proto_file := strings.Replace(config_file, "json", "proto", -1)

	fmt.Fprintf(os.Stdout, "proto_dest_path: %v    proto_file: %v\n", proto_dest_path, proto_file)

	if !config_loader.GenerateFieldStructsProto(proto_dest_path + proto_file) {
		fmt.Fprintf(os.Stderr, "generate proto file failed\n")
		return
	}

	fmt.Fprintf(os.Stdout, "generated proto\n")

	cmd := exec.Command(protoc_path, "--go_out", dest_path /*+"/"+config_loader.DBPkg*/, "--proto_path", proto_dest_path, proto_file)
	var out bytes.Buffer

	fmt.Fprintf(os.Stdout, "--go_out=%v  --proto_path=%v  proto_file=%v\n", dest_path /*+"/"+config_loader.DBPkg*/, proto_dest_path, proto_file)

	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "cmd run err: %v\n", err.Error())
		return
	}
	fmt.Printf("%s", out.String())

	if !config_loader.GenerateInitFunc(dest_path) {
		fmt.Fprintf(os.Stderr, "generate init func failed\n")
		return
	}

	fmt.Fprintf(os.Stdout, "generated init funcs\ngenerated all\n")
}
)))
\"fweageawfewafewa
package mysql_generate

import (
	"log"
	"os"
	"strings"

	mysql_base "github.com/huoshan017/mysql-go/base"
)

func _upper_first_char(str string) string {
	if str == "" {
		return str
	}
	c := []byte(str)
	var uppered bool
	for i := 0; i < len(c); i++ {
		if i == 0 || c[i-1] == '_' {
			if int32(c[i]) >= int32('a') && int32(c[i]) <= int32('z') {
				c[i] = byte(int32(c[i]) + int32('A') - int32('a'))
				uppered = true
			}
		}
	}
	if !uppered {
		return str
	}
	return string(c)
}

func gen_row_func(struct_row_name string, go_type string, field *mysql_base.FieldConfig) string {
	var str string
	str += "func (this *" + struct_row_name + ") Get_" + field.Name + "() " + go_type + " {\n"
	str += "	return this." + field.Name + "\n"
	str += "}\n\n"
	str += "func (this *" + struct_row_name + ") Set_" + field.Name + "(v " + go_type + ") {\n"
	str += "	this." + field.Name + " = v\n"
	str += "}\n\n"
	str += "func (this *" + struct_row_name + ") Get_" + field.Name + "_WithLock() " + go_type + " {\n"
	str += "	this.locker.RLock()\n"
	str += "	defer this.locker.RUnlock()\n"
	str += "	return this." + field.Name + "\n"
	str += "}\n\n"
	str += "func (this *" + struct_row_name + ") Set_" + field.Name + "_WithLock(v " + go_type + ") {\n"
	str += "	this.locker.Lock()\n"
	str += "	defer this.locker.Unlock()\n"
	str += "	this." + field.Name + " = v\n"
	str += "}\n\n"
	if field.StructName != "" {
		str += "func (this *" + struct_row_name + ") Marshal_" + field.Name + "() []byte {\n"
		str += "	if this." + field.Name + " == nil {\n"
		str += "		return nil\n"
		str += "	}\n"
		str += "	data, err := proto.Marshal(this." + field.Name + ")\n"
		str += "	if err != nil {\n"
		str += "		log.Printf(\"Marshal " + field.StructName + " failed err(%v)!\\n\", err.Error())\n"
		str += "		return nil\n"
		str += "	}\n"
		str += "	return data\n"
		str += "}\n\n"
		str += "func (this *" + struct_row_name + ") Unmarshal_" + field.Name + "(data []byte) bool {\n"
		str += "	err := proto.Unmarshal(data, this." + field.Name + ")\n"
		str += "	if err != nil {\n"
		str += "		log.Printf(\"Unmarshal " + field.StructName + " failed err(%v)!\\n\", err.Error())\n"
		str += "		return false\n"
		str += "	}\n"
		str += "	return true\n"
		str += "}\n\n"
		str += "func (this *" + struct_row_name + ") Get_" + field.Name + "_FVP() *mysql_base.FieldValuePair {\n"
		str += "	data := this.Marshal_" + field.Name + "()\n"
		str += "	if data == nil {\n"
		str += "		return nil\n"
		str += "	}\n"
		str += "	return &mysql_base.FieldValuePair{ Name: \"" + field.Name + "\", Value: data }\n"
		str += "}\n\n"
	} else {
		str += "func (this *" + struct_row_name + ") Get_" + field.Name + "_FVP() *mysql_base.FieldValuePair {\n"
		str += "	return &mysql_base.FieldValuePair{ Name: \"" + field.Name + "\", Value: this.Get_" + field.Name + "() }\n"
		str += "}\n\n"
	}
	return str
}

func gen_row_get_fvp_list_with_name_func(struct_row_name, field_func_map string) string {
	var str string
	str += "func (this *" + struct_row_name + ") GetFVPList(fields_name []string) []*mysql_base.FieldValuePair {\n"
	str += "	var field_list []*mysql_base.FieldValuePair\n"
	str += "	for _, field_name := range fields_name {\n"
	str += "		fun := " + field_func_map + "[field_name]\n"
	str += "		if fun == nil {\n"
	str += "			continue\n"
	str += "		}\n"
	str += "		value_pair := fun(this)\n"
	str += "		if value_pair != nil {\n"
	str += "			field_list = append(field_list, value_pair)\n"
	str += "		}\n"
	str += "	}\n"
	str += "	return field_list\n"
	str += "}\n\n"
	return str
}

func gen_row_format_all_fvp_func(struct_row_name string, table *mysql_base.TableConfig) string {
	var str string
	str += ("func (this *" + struct_row_name + ") _format_field_list() []*mysql_base.FieldValuePair {\n")
	str += ("	var field_list []*mysql_base.FieldValuePair\n")
	for _, field := range table.Fields {
		/*is_unsigned := strings.Contains(strings.ToLower(field.Type), "unsigned")
		if mysql_base.MysqlFieldTypeStr2GoTypeStr(strings.ToUpper(field.Type), is_unsigned) == "" {
			continue
		}*/
		if strings.Contains(strings.ToLower(field.Type), "timestamp") {
			continue
		}

		if field.StructName != "" {
			str += "	data_" + field.Name + " := this.Marshal_" + field.Name + "()\n"
			str += "	if data_" + field.Name + " != nil {\n"
			str += "		field_list = append(field_list, &mysql_base.FieldValuePair{ Name: \"" + field.Name + "\", Value: data_" + field.Name + " })\n"
			str += "	}\n"
		} else {
			str += "	field_list = append(field_list, &mysql_base.FieldValuePair{ Name: \"" + field.Name + "\", Value: this.Get_" + field.Name + "() })\n"
		}
	}
	str += "	return field_list\n"
	str += "}\n\n"
	return str
}

func gen_row_lock_func(struct_row_name string) string {
	var str string
	str += "func (this *" + struct_row_name + ") Lock() {\n"
	str += "	this.locker.Lock()\n"
	str += "}\n\n"
	str += "func (this *" + struct_row_name + ") Unlock() {\n"
	str += "	this.locker.Unlock()\n"
	str += "}\n\n"
	str += "func (this *" + struct_row_name + ") RLock() {\n"
	str += "	this.locker.RLock()\n"
	str += "}\n\n"
	str += "func (this *" + struct_row_name + ") RUnlock() {\n"
	str += "	this.locker.RUnlock()\n"
	str += "}\n\n"
	var row_atomic_exec_func = struct_row_name + "_AtomicExecFunc"
	str += "type " + row_atomic_exec_func + " func(*" + struct_row_name + ")\n\n"
	str += "func (this *" + struct_row_name + ") AtomicExecute(exec_func " + row_atomic_exec_func + ") {\n"
	str += "	this.locker.Lock()\n"
	str += "	defer this.locker.Unlock()\n"
	str += "	exec_func(this)\n"
	str += "}\n\n"
	str += "func (this *" + struct_row_name + ") AtomicExecuteReadOnly(exec_func " + row_atomic_exec_func + ") {\n"
	str += "	this.locker.RLock()\n"
	str += "	defer this.locker.RUnlock()\n"
	str += "	exec_func(this)\n"
	str += "}\n\n"
	return str
}

func gen_get_result_list(table *mysql_base.TableConfig, struct_row_name, bytes_define_list, dest_list string) (str string) {
	str = ("	var r []*" + struct_row_name + "\n")
	if bytes_define_list != "" {
		str += ("	var " + bytes_define_list + " []byte\n")
	}
	str += ("	for {\n")
	str += ("		var t = Create" + struct_row_name + "()\n")
	str += ("		var dest_list = []any{" + dest_list + "}\n")
	str += ("		if !result_list.Get(dest_list...) {\n")
	str += ("			break\n")
	str += ("		}\n")
	for _, field := range table.Fields {
		if field.StructName != "" && (mysql_base.IsMysqlFieldBinaryType(field.RealType) || mysql_base.IsMysqlFieldBlobType(field.RealType)) {
			str += "		t.Unmarshal_" + field.Name + "(data_" + field.Name + ")\n"
		}
	}
	str += ("		r = append(r, t)\n")
	str += ("	}\n")
	return
}

func gen_get_result_map(table *mysql_base.TableConfig, struct_row_name, bytes_define_list, dest_list, primary_field_type string) (str string) {
	str = ("	var r = make(map[" + primary_field_type + "]*" + struct_row_name + ")\n")
	if bytes_define_list != "" {
		str += ("	var " + bytes_define_list + " []byte\n")
	}
	str += ("	for {\n")
	str += ("		var t = Create" + struct_row_name + "()\n")
	str += ("		var dest_list = []any{" + dest_list + "}\n")
	str += ("		if !result_list.Get(dest_list...) {\n")
	str += ("			break\n")
	str += ("		}\n")
	for _, field := range table.Fields {
		if field.StructName != "" && (mysql_base.IsMysqlFieldBinaryType(field.RealType) || mysql_base.IsMysqlFieldBlobType(field.RealType)) {
			str += "		t.Unmarshal_" + field.Name + "(data_" + field.Name + ")\n"
		}
	}
	str += ("		r[t." + table.PrimaryKey + "] = t\n")
	str += ("	}\n")
	return
}

func gen_source(f *os.File, pkg_name string, table *mysql_base.TableConfig) bool {
	str := "package " + pkg_name + "\n\nimport (\n"
	if table.HasStructField() {
		str += "	\"log\"\n"
	}
	str += "	\"sync\"\n"
	if !table.SingleRow {
		str += "	\"github.com/hashicorp/golang-lru/simplelru\"\n"
	}
	str += "	\"github.com/huoshan017/mysql-go/base\"\n"
	str += "	\"github.com/huoshan017/mysql-go/manager\"\n"
	str += "	\"github.com/huoshan017/mysql-go/proxy/client\"\n"
	if table.HasStructField() {
		str += "	\"github.com/golang/protobuf/proto\"\n"
	}
	str += ")\n\n"

	struct_row_name := _upper_first_char(table.Name)
	struct_table_name := struct_row_name + "Table"

	field_pair_func_define := table.Name + "_field_pair_func"
	field_pair_func_type := "func (t *" + struct_row_name + ") *mysql_base.FieldValuePair"
	str += "type " + field_pair_func_define + " " + field_pair_func_type + "\n\n"

	field_func_map := table.Name + "_fields_map"
	str += "var " + field_func_map + " = map[string]" + field_pair_func_define + "{\n"
	for _, field := range table.Fields {
		is_unsigned := strings.Contains(strings.ToLower(field.Type), "unsigned")
		if mysql_base.MysqlFieldTypeStr2GoTypeStr(strings.ToUpper(field.Type), is_unsigned) == "" {
			continue
		}
		str += "	\"" + field.Name + "\": " + field_pair_func_type + "{\n"
		str += "		return t.Get_" + field.Name + "_FVP()\n"
		str += "	},\n"
	}
	str += "}\n\n"

	// row struct
	var init_mem_list, row_func_list string
	str += ("type " + struct_row_name + " struct {\n")
	for _, field := range table.Fields {
		var go_type string
		if field.StructName != "" {
			go_type = "*" + field.StructName
			init_mem_list += "		" + field.Name + ": &" + field.StructName + "{},\n"
		} else {
			is_unsigned := strings.Contains(strings.ToLower(field.Type), "unsigned")
			go_type = mysql_base.MysqlFieldType2GoTypeStr(field.RealType, is_unsigned)
			if go_type == "" {
				log.Printf("get go type failed by field type %v in table %v, to continue\n", field.Type, table.Name)
				continue
			}
		}
		str += ("	" + field.Name + " " + go_type + "\n")
		row_func_list += gen_row_func(struct_row_name, go_type, field)
	}
	str += "	locker sync.RWMutex\n"
	str += "}\n\n"
	str += "func Create" + struct_row_name + "() *" + struct_row_name + " {\n"
	str += "	return &" + struct_row_name + "{\n"
	if init_mem_list != "" {
		str += init_mem_list
	}
	str += "	}\n"
	str += "}\n\n"
	str += row_func_list
	str += gen_row_get_fvp_list_with_name_func(struct_row_name, field_func_map)
	str += gen_row_format_all_fvp_func(struct_row_name, table)
	str += gen_row_lock_func(struct_row_name)

	// table
	str += ("type " + struct_table_name + " struct {\n")
	str += "	db *mysql_manager.DB\n"
	if table.SingleRow {
		str += "	row *" + struct_row_name + "\n"
	}
	str += "}\n\n"

	// init func
	str += ("func (this *" + struct_table_name + ") Init(db *mysql_manager.DB) {\n")
	str += ("	this.db = db\n")
	str += "}\n\n"

	var field_list string
	for i, field := range table.Fields {
		is_unsigned := strings.Contains(strings.ToLower(field.Type), "unsigned")
		go_type := mysql_base.MysqlFieldType2GoTypeStr(field.RealType, is_unsigned)
		if go_type == "" {
			continue
		}
		if i == 0 {
			field_list = "\"" + field.Name + "\""
		} else {
			field_list += (", \"" + field.Name + "\"")
		}
	}

	var bytes_define_list string
	var dest_list string
	for _, field := range table.Fields {
		is_unsigned := strings.Contains(strings.ToLower(field.Type), "unsigned")
		go_type := mysql_base.MysqlFieldType2GoTypeStr(field.RealType, is_unsigned)
		if go_type == "" {
			continue
		}

		var dest string
		if field.StructName != "" && (mysql_base.IsMysqlFieldBinaryType(field.RealType) || mysql_base.IsMysqlFieldBlobType(field.RealType)) {
			dest = "data_" + field.Name
			if bytes_define_list == "" {
				bytes_define_list = dest
			} else {
				bytes_define_list += (", " + dest)
			}
		} else {
			dest = "t." + field.Name
		}

		if dest_list == "" {
			dest_list = "&" + dest
		} else {
			dest_list += (", &" + dest)
		}
	}

	// select func
	if !table.SingleRow {
		str += ("func (this *" + struct_table_name + ") Select(field_name string, field_value any) (*" + struct_row_name + ", error) {\n")
	} else {
		str += "func (this *" + struct_table_name + ") Select() (*" + struct_row_name + ", error) {\n"
	}
	str += ("	var field_list = []string{" + field_list + "}\n")
	str += ("	var t = Create" + struct_row_name + "()\n")
	if bytes_define_list != "" {
		str += ("	var " + bytes_define_list + " []byte\n")
	}
	str += ("	var err error\n")
	str += ("	var dest_list = []any{" + dest_list + "}\n")
	if !table.SingleRow {
		str += ("	err = this.db.Select(\"" + table.Name + "\", field_name, field_value, field_list, dest_list)\n")
	} else {
		str += ("	err = this.db.Select(\"" + table.Name + "\", \"place_hold\", 1, field_list, dest_list)\n")
	}
	str += ("	if err != nil {\n")
	str += ("		return nil, err\n")
	str += ("	}\n")
	for _, field := range table.Fields {
		if field.StructName != "" && (mysql_base.IsMysqlFieldBinaryType(field.RealType) || mysql_base.IsMysqlFieldBlobType(field.RealType)) {
			str += "	t.Unmarshal_" + field.Name + "(data_" + field.Name + ")\n"
		}
	}
	str += ("	return t, nil\n")
	str += ("}\n\n")

	// primary field
	var pf *mysql_base.FieldConfig
	var pt string
	if !table.SingleRow {
		pf = table.GetPrimaryKeyFieldConfig()
		if pf == nil {
			log.Printf("cant get table %v primary key\n", table.Name)
			return false
		}
		if !(mysql_base.IsMysqlFieldIntType(pf.RealType) || mysql_base.IsMysqlFieldTextType(pf.RealType)) {
			log.Printf("not support primary type %v for table %v", pf.Type, table.Name)
			return false
		}
		is_unsigned := strings.Contains(strings.ToLower(pf.Type), "unsigned")
		pt = mysql_base.MysqlFieldType2GoTypeStr(pf.RealType, is_unsigned)
		if pt == "" {
			log.Printf("主键类型%v没有对应的数据类型\n")
			return false
		}
	}

	if !table.SingleRow {
		// select records count
		str += "func (this *" + struct_table_name + ") SelectRecordsCount() (count int32, err error) {\n"
		str += "	return this.db.SelectRecordsCount(\"" + table.Name + "\")\n"
		str += "}\n\n"

		// select records count by field
		str += "func (this *" + struct_table_name + ") SelectRecordsCountByField(field_name string, field_value any) (count int32, err error) {\n"
		str += "	return this.db.SelectRecordsCountByField(\"" + table.Name + "\", field_name, field_value)\n"
		str += "}\n\n"

		// select records condition
		str += "func (this *" + struct_table_name + ") SelectRecordsCondition(field_name string, field_value any, sel_cond *mysql_base.SelectCondition) ([]*" + struct_row_name + ", error) {\n"
		str += "	var field_list = []string{" + field_list + "}\n"
		str += "	var result_list mysql_base.QueryResultList\n"
		str += "	err := this.db.SelectRecordsCondition(\"" + table.Name + "\", field_name, field_value, sel_cond, field_list, &result_list)\n"
		str += "	if err != nil {\n"
		str += "		return nil, err\n"
		str += "	}\n"
		str += gen_get_result_list(table, struct_row_name, bytes_define_list, dest_list)
		str += "	return r, nil\n"
		str += "}\n\n"

		// select records map condition
		str += "func (this *" + struct_table_name + ") SelectRecordsMapCondition(field_name string, field_value any, sel_cond *mysql_base.SelectCondition) (map[" + pt + "]*" + struct_row_name + ", error) {\n"
		str += "	var field_list = []string{" + field_list + "}\n"
		str += "	var result_list mysql_base.QueryResultList\n"
		str += "	err := this.db.SelectRecordsCondition(\"" + table.Name + "\", field_name, field_value, sel_cond, field_list, &result_list)\n"
		str += "	if err != nil {\n"
		str += "		return nil, err\n"
		str += "	}\n"
		str += gen_get_result_map(table, struct_row_name, bytes_define_list, dest_list, pt)
		str += "	return r, nil\n"
		str += "}\n\n"

		// select all records
		str += "func (this *" + struct_table_name + ") SelectAllRecords() ([]*" + struct_row_name + ", error) {\n"
		str += "	var field_list = []string{" + field_list + "}\n"
		str += "	var result_list mysql_base.QueryResultList\n"
		str += "	err := this.db.SelectAllRecords(\"" + table.Name + "\", field_list, &result_list)\n"
		str += "	if err != nil {\n"
		str += "		return nil, err\n"
		str += "	}\n"
		str += gen_get_result_list(table, struct_row_name, bytes_define_list, dest_list)
		str += "	return r, nil\n"
		str += "}\n\n"

		// select all map records
		str += "func (this *" + struct_table_name + ") SelectAllMapRecords() (map[" + pt + "]*" + struct_row_name + ", error) {\n"
		str += "	var field_list = []string{" + field_list + "}\n"
		str += "	var result_list mysql_base.QueryResultList\n"
		str += "	err := this.db.SelectAllRecords(\"" + table.Name + "\", field_list, &result_list)\n"
		str += "	if err != nil {\n"
		str += "		return nil, err\n"
		str += "	}\n"
		str += gen_get_result_map(table, struct_row_name, bytes_define_list, dest_list, pt)
		str += "	return r, nil\n"
		str += "}\n\n"

		// select records
		str += "func (this *" + struct_table_name + ") SelectRecords(field_name string, field_value any) ([]*" + struct_row_name + ", error) {\n"
		str += "	var field_list = []string{" + field_list + "}\n"
		str += "	var result_list mysql_base.QueryResultList\n"
		str += "	err := this.db.SelectRecords(\"" + table.Name + "\", field_name, field_value, field_list, &result_list)\n"
		str += "	if err != nil {\n"
		str += "		return nil, err\n"
		str += "	}\n"
		str += gen_get_result_list(table, struct_row_name, bytes_define_list, dest_list)
		str += "	return r, nil\n"
		str += "}\n\n"

		// select records map
		str += "func (this *" + struct_table_name + ") SelectRecordsMap(field_name string, field_value any) (map[" + pt + "]*" + struct_row_name + ", error) {\n"
		str += "	var field_list = []string{" + field_list + "}\n"
		str += "	var result_list mysql_base.QueryResultList\n"
		str += "	err := this.db.SelectRecords(\"" + table.Name + "\", field_name, field_value, field_list, &result_list)\n"
		str += "	if err != nil {\n"
		str += "		return nil, err\n"
		str += "	}\n"
		str += gen_get_result_map(table, struct_row_name, bytes_define_list, dest_list, pt)
		str += "	return r, nil\n"
		str += "}\n\n"

		// select primary field
		str += ("func (this *" + struct_table_name + ") SelectByPrimaryField(key " + pt + ") (*" + struct_row_name + ", error) {\n")
		str += ("	v, err := this.Select(\"" + pf.Name + "\", key)\n")
		str += ("	if err != nil {\n")
		str += ("		return nil, err\n")
		str += ("	}\n")
		str += ("	return v, nil\n")
		str += ("}\n\n")

		// select all primary field
		str += ("func (this *" + struct_table_name + ") SelectAllPrimaryField() ([]" + pt + ", error) {\n")
		str += ("	var result_list mysql_base.QueryResultList\n")
		str += ("	err := this.db.SelectFieldNoKey(\"" + table.Name + "\", \"" + pf.Name + "\", &result_list)\n")
		str += ("	if err != nil {\n")
		str += ("		return nil, err\n")
		str += ("	}\n")
		str += ("	var value_list []" + pt + "\n")
		str += ("	for {\n")
		str += ("		var d " + pt + "\n")
		str += ("		if !result_list.Get(&d) {\n")
		str += ("			break\n")
		str += ("		}\n")
		str += ("		value_list = append(value_list, d)\n")
		str += ("	}\n")
		str += ("	return value_list, nil\n")
		str += ("}\n\n")

		// select all primary field map
		str += ("func (this *" + struct_table_name + ") SelectAllPrimaryFieldMap() (map[" + pt + "]bool, error) {\n")
		str += ("	var result_list mysql_base.QueryResultList\n")
		str += ("	err := this.db.SelectFieldNoKey(\"" + table.Name + "\", \"" + pf.Name + "\", &result_list)\n")
		str += ("	if err != nil {\n")
		str += ("		return nil, err\n")
		str += ("	}\n")
		str += ("	var value_map = make(map[" + pt + "]bool)\n")
		str += ("	for {\n")
		str += ("		var d " + pt + "\n")
		str += ("		if !result_list.Get(&d) {\n")
		str += ("			break\n")
		str += ("		}\n")
		str += ("		value_map[d] = true\n")
		str += ("	}\n")
		str += ("	return value_map, nil\n")
		str += ("}\n\n")

		// insert
		str += "func (this *" + struct_table_name + ") Insert(t *" + struct_row_name + ") {\n"
		str += "	var field_list = t._format_field_list()\n"
		str += "	if field_list != nil {\n"
		str += "		this.db.Insert(\"" + table.Name + "\", field_list)\n"
		str += "	}\n"
		str += "}\n\n"

		// insert ignore
		str += "func (this *" + struct_table_name + ") InsertIgnore(t *" + struct_row_name + ") {\n"
		str += "	var field_list = t._format_field_list()\n"
		str += "	if field_list != nil {\n"
		str += "		this.db.InsertIgnore(\"" + table.Name + "\", field_list)\n"
		str += "	}\n"
		str += "}\n\n"

		// delete
		str += ("func (this *" + struct_table_name + ") Delete(" + pf.Name + " " + pt + ") {\n")
		str += ("	this.db.Delete(\"" + table.Name + "\", \"" + pf.Name + "\", " + pf.Name + ")\n")
		str += ("}\n\n")

		// create row func
		str += "func (this *" + struct_table_name + ") NewRecord(" + pf.Name + " " + pt + ") *" + struct_row_name + " {\n"
		str += "	return &" + struct_row_name + "{ " + pf.Name + ": " + pf.Name + ", }\n"
		str += "}\n\n"
	} else {
		str += "func (this *" + struct_table_name + ") GetRow() (*" + struct_row_name + ", error) {\n"
		str += "	if this.row == nil {\n"
		str += "		row, err := this.Select()\n"
		str += "		if err != nil {\n"
		str += "			return nil, err\n"
		str += "		}\n"
		str += "		this.row = row\n"
		str += "	}\n"
		str += "	return this.row, nil\n"
		str += "}\n\n"
	}

	// update
	str += "func (this *" + struct_table_name + ") UpdateAll(t *" + struct_row_name + ") {\n"
	str += "	var field_list = t._format_field_list()\n"
	str += "	if field_list != nil {\n"
	if !table.SingleRow {
		str += "		this.db.Update(\"" + table.Name + "\", \"" + pf.Name + "\", t.Get_" + pf.Name + "(), field_list)\n"
	} else {
		str += "		this.db.Update(\"" + table.Name + "\", \"place_hold\", 1, field_list)\n"
	}
	str += "	}\n"
	str += "}\n\n"

	// update some field
	if !table.SingleRow {
		str += "func (this *" + struct_table_name + ") UpdateWithFVPList(" + pf.Name + " " + pt + ", field_list []*mysql_base.FieldValuePair) {\n"
		str += "	this.db.Update(\"" + table.Name + "\", \"" + pf.Name + "\", " + pf.Name + ", field_list)\n"
	} else {
		str += "func (this *" + struct_table_name + ") UpdateWithFVPList(field_list []*mysql_base.FieldValuePair) {\n"
		str += "	this.db.Update(\"" + table.Name + "\", \"place_hold\", 1, field_list)\n"
	}
	str += "}\n\n"

	// update by field name
	str += "func (this *" + struct_table_name + ") UpdateWithFieldName(t *" + struct_row_name + ", fields_name []string) {\n"
	str += "	var field_list = t.GetFVPList(fields_name)\n"
	str += "	if field_list != nil {\n"
	if !table.SingleRow {
		str += "		this.UpdateWithFVPList(t.Get_" + pf.Name + "(), field_list)\n"
	} else {
		str += "		this.UpdateWithFVPList(field_list)\n"
	}
	str += "	}\n"
	str += "}\n\n"

	str += gen_procedure_source(table, struct_table_name, struct_row_name, pf, pt)

	_, err := f.WriteString(str)
	if err != nil {
		log.Printf("write string err %v\n", err.Error())
		return false
	}

	return true
}

func gen_procedure_source(table *mysql_base.TableConfig, struct_table_name, struct_row_name string, primary_field *mysql_base.FieldConfig, primary_type string) string {
	var str string

	if !table.SingleRow {
		str += "func (this *" + struct_table_name + ") TransactionInsert(transaction *mysql_manager.Transaction, t *" + struct_row_name + ") {\n"
		str += "	field_list := t._format_field_list()\n"
		str += "	if field_list != nil {\n"
		str += "		transaction.Insert(\"" + table.Name + "\", field_list)\n"
		str += "	}\n"
		str += "}\n\n"
		str += "func (this *" + struct_table_name + ") TransactionDelete(transaction *mysql_manager.Transaction, " + primary_field.Name + " " + primary_type + ") {\n"
		str += "	transaction.Delete(\"" + table.Name + "\", \"" + primary_field.Name + "\", " + primary_field.Name + ")\n"
		str += "}\n\n"
	}

	str += "func (this *" + struct_table_name + ") TransactionUpdateAll(transaction *mysql_manager.Transaction, t*" + struct_row_name + ") {\n"
	str += "	field_list := t._format_field_list()\n"
	str += "	if field_list != nil {\n"
	if !table.SingleRow {
		str += "		transaction.Update(\"" + table.Name + "\", \"" + primary_field.Name + "\", t.Get_" + primary_field.Name + "(), field_list)\n"
	} else {
		str += "		transaction.Update(\"" + table.Name + "\", \"place_hold\", 1, field_list)\n"
	}
	str += "	}\n"
	str += "}\n\n"

	if !table.SingleRow {
		str += "func (this *" + struct_table_name + ") TransactionUpdateWithFVPList(transaction *mysql_manager.Transaction, " + primary_field.Name + " " + primary_type + ", field_list []*mysql_base.FieldValuePair) {\n"
		str += "	transaction.Update(\"" + table.Name + "\", \"" + primary_field.Name + "\", " + primary_field.Name + ", field_list)\n"
	} else {
		str += "func (this *" + struct_table_name + ") TransactionUpdateWithFVPList(transaction *mysql_manager.Transaction, field_list []*mysql_base.FieldValuePair) {\n"
		str += "	transaction.Update(\"" + table.Name + "\", \"place_hold\", 1, field_list)\n"
	}
	str += "}\n\n"

	str += "func (this *" + struct_table_name + ") TransactionUpdateWithFieldName(transaction *mysql_manager.Transaction, t *" + struct_row_name + ", fields_name []string) {\n"
	str += "	field_list := t.GetFVPList(fields_name)\n"
	str += "	if field_list != nil {\n"
	if !table.SingleRow {
		str += "		transaction.Update(\"" + table.Name + "\", \"" + primary_field.Name + "\", t.Get_" + primary_field.Name + "(), field_list)\n"
	} else {
		str += "		transaction.Update(\"" + table.Name + "\", \"place_hold\", 1, field_list)\n"
	}
	str += "	}\n"
	str += "}\n\n"

	return str
}
package mysql_proxy_common

import (
	"bufio"
	"encoding/gob"
	"errors"
	"io"
	"log"
	"net"
	"reflect"
	"runtime/debug"
	"strings"
	"sync"
	"unicode"
	"unicode/utf8"
)

const (
	// Defaults used by HandleHTTP
	DefaultRPCPath   = "/_goRPC_"
	DefaultDebugPath = "/debug/rpc"
	debugLog         = false
)

// Precompute the reflect type for error. Can't use error directly
// because Typeof takes an empty interface value. This is annoying.
var typeOfError = reflect.TypeOf((*error)(nil)).Elem()

type methodType struct {
	sync.Mutex // protects counters
	method     reflect.Method
	ArgType    reflect.Type
	ReplyType  reflect.Type
	numCalls   uint
}

const (
	RECEIVE_LIST_DEFAULT_LENGTH = 1000
)

type service struct {
	name   string                 // name of service
	rcvr   reflect.Value          // receiver of methods for the service
	typ    reflect.Type           // type of the receiver
	method map[string]*methodType // registered methods
}

// Request is a header written before every RPC call. It is used internally
// but documented here as an aid to debugging, such as when analyzing
// network traffic.
type Request struct {
	ServiceMethod string   // format: "Service.Method"
	Seq           uint64   // sequence number chosen by client
	next          *Request // for free list in Server
}

// Response is a header written before every RPC return. It is used internally
// but documented here as an aid to debugging, such as when analyzing
// network traffic.
type Response struct {
	ServiceMethod string    // echoes that of the Request
	Seq           uint64    // echoes that of the request
	Error         string    // error, if any.
	next          *Response // for free list in Server
}

// Server represents an RPC Server.
type Server struct {
	serviceMap      sync.Map   // map[string]*service
	reqLock         sync.Mutex // protects freeReq
	freeReq         *Request
	respLock        sync.Mutex // protects freeResp
	freeResp        *Response
	isOnlyRecvCache bool // 是否缓存只读goroutine的接收数据
}

// NewServer returns a new Server.
func NewServer(only_recv_cache bool) *Server {
	return &Server{
		isOnlyRecvCache: only_recv_cache,
	}
}

// DefaultServer is the default instance of *Server.
var DefaultServer = NewServer(false)

// Is this an exported - upper case - name?
func isExported(name string) bool {
	rune, _ := utf8.DecodeRuneInString(name)
	return unicode.IsUpper(rune)
}

// Is this type exported or a builtin?
func isExportedOrBuiltinType(t reflect.Type) bool {
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	// PkgPath will be non-empty even for an exported type,
	// so we need to check the type name as well.
	return isExported(t.Name()) || t.PkgPath() == ""
}

// Register publishes in the server the set of methods of the
// receiver value that satisfy the following conditions:
//	- exported method of exported type
//	- two arguments, both of exported type
//	- the second argument is a pointer
//	- one return value, of type error
// It returns an error if the receiver is not an exported type or has
// no suitable methods. It also logs the error using package log.
// The client accesses each method using a string of the form "Type.Method",
// where Type is the receiver's concrete type.
func (server *Server) Register(rcvr any) error {
	return server.register(rcvr, "", false)
}

// RegisterName is like Register but uses the provided name for the type
// instead of the receiver's concrete type.
func (server *Server) RegisterName(name string, rcvr any) error {
	return server.register(rcvr, name, true)
}

func (server *Server) register(rcvr any, name string, useName bool) error {
	s := new(service)
	s.typ = reflect.TypeOf(rcvr)
	s.rcvr = reflect.ValueOf(rcvr)
	sname := reflect.Indirect(s.rcvr).Type().Name()
	if useName {
		sname = name
	}
	if sname == "" {
		s := "rpc.Register: no service name for type " + s.typ.String()
		log.Print(s)
		return errors.New(s)
	}
	if !isExported(sname) && !useName {
		s := "rpc.Register: type " + sname + " is not exported"
		log.Print(s)
		return errors.New(s)
	}
	s.name = sname

	// Install the methods
	s.method = suitableMethods(s.typ, true)

	if len(s.method) == 0 {
		str := ""

		// To help the user, see if a pointer receiver would work.
		method := suitableMethods(reflect.PtrTo(s.typ), false)
		if len(method) != 0 {
			str = "rpc.Register: type " + sname + " has no exported methods of suitable type (hint: pass a pointer to value of that type)"
		} else {
			str = "rpc.Register: type " + sname + " has no exported methods of suitable type"
		}
		log.Print(str)
		return errors.New(str)
	}

	if _, dup := server.serviceMap.LoadOrStore(sname, s); dup {
		return errors.New("rpc: service already defined: " + sname)
	}
	return nil
}

// suitableMethods returns suitable Rpc methods of typ, it will report
// error using log if reportErr is true.
func suitableMethods(typ reflect.Type, reportErr bool) map[string]*methodType {
	methods := make(map[string]*methodType)
	for m := 0; m < typ.NumMethod(); m++ {
		method := typ.Method(m)
		mtype := method.Type
		mname := method.Name
		// Method must be exported.
		if method.PkgPath != "" {
			continue
		}
		// Method needs three ins: receiver, *args, *reply.
		if mtype.NumIn() != 3 {
			if reportErr {
				log.Printf("rpc.Register: method %q has %d input parameters; needs exactly three\n", mname, mtype.NumIn())
			}
			continue
		}
		// First arg need not be a pointer.
		argType := mtype.In(1)
		if !isExportedOrBuiltinType(argType) {
			if reportErr {
				log.Printf("rpc.Register: argument type of method %q is not exported: %q\n", mname, argType)
			}
			continue
		}
		// Second arg must be a pointer.
		replyType := mtype.In(2)
		if replyType.Kind() != reflect.Ptr {
			if reportErr {
				log.Printf("rpc.Register: reply type of method %q is not a pointer: %q\n", mname, replyType)
			}
			continue
		}
		// Reply type must be exported.
		if !isExportedOrBuiltinType(replyType) {
			if reportErr {
				log.Printf("rpc.Register: reply type of method %q is not exported: %q\n", mname, replyType)
			}
			continue
		}
		// Method needs one out.
		if mtype.NumOut() != 1 {
			if reportErr {
				log.Printf("rpc.Register: method %q has %d output parameters; needs exactly one\n", mname, mtype.NumOut())
			}
			continue
		}
		// The return type of the method must be error.
		if returnType := mtype.Out(0); returnType != typeOfError {
			if reportErr {
				log.Printf("rpc.Register: return type of method %q is %q, must be error\n", mname, returnType)
			}
			continue
		}
		methods[mname] = &methodType{method: method, ArgType: argType, ReplyType: replyType}
	}
	return methods
}

// A value sent as a placeholder for the server's response value when the server
// receives an invalid request. It is never decoded by the client since the Response
// contains an error when it is used.
var invalidRequest = struct{}{}

func (server *Server) sendResponse(sending *sync.Mutex, req *Request, reply any, codec ServerCodec, errmsg string) {
	resp := server.getResponse()
	// Encode the response header
	resp.ServiceMethod = req.ServiceMethod
	if errmsg != "" {
		resp.Error = errmsg
		reply = invalidRequest
	}
	resp.Seq = req.Seq
	sending.Lock()
	err := codec.WriteResponse(resp, reply)
	if debugLog && err != nil {
		log.Println("rpc: writing response:", err)
	}
	sending.Unlock()
	server.freeResponse(resp)
}

func (m *methodType) NumCalls() (n uint) {
	m.Lock()
	n = m.numCalls
	m.Unlock()
	return n
}

func (s *service) call(server *Server, sending *sync.Mutex, wg *sync.WaitGroup, mtype *methodType, req *Request, argv, replyv reflect.Value, codec ServerCodec) {
	if wg != nil {
		defer wg.Done()
	}
	mtype.Lock()
	mtype.numCalls++
	mtype.Unlock()
	function := mtype.method.Func
	// Invoke the method, providing a new value for the reply.
	returnValues := function.Call([]reflect.Value{s.rcvr, argv, replyv})
	// The return value for the method is an error.
	errInter := returnValues[0].Interface()
	errmsg := ""
	if errInter != nil {
		errmsg = errInter.(error).Error()
	}
	server.sendResponse(sending, req, replyv.Interface(), codec, errmsg)
	server.freeRequest(req)
}

func (s *service) call_only_recv(server *Server, mtype *methodType, req *Request, argv, replyv reflect.Value, codec ServerCodec) {
	mtype.Lock()
	mtype.numCalls++
	mtype.Unlock()
	function := mtype.method.Func
	// Invoke the method, providing a new value for the reply.
	returnValues := function.Call([]reflect.Value{s.rcvr, argv, replyv})
	// The return value for the method is an error.
	errInter := returnValues[0].Interface()
	errmsg := ""
	if errInter != nil {
		errmsg = errInter.(error).Error()
		log.Println("rpc.Serve ", errmsg)
	}
	server.freeRequest(req)
}

type receive_data struct {
	srv    *service
	mtype  *methodType
	req    *Request
	argv   reflect.Value
	replyv reflect.Value
}

type gobServerCodec struct {
	rwc             io.ReadWriteCloser
	dec             *gob.Decoder
	enc             *gob.Encoder
	encBuf          *bufio.Writer
	closed          bool
	onlyReceiveChan chan *receive_data // 只接收缓冲区
}

func (c *gobServerCodec) ReadRequestHeader(r *Request) error {
	return c.dec.Decode(r)
}

func (c *gobServerCodec) ReadRequestBody(body any) error {
	return c.dec.Decode(body)
}

func (c *gobServerCodec) WriteResponse(r *Response, body any) (err error) {
	if err = c.enc.Encode(r); err != nil {
		if c.encBuf.Flush() == nil {
			// Gob couldn't encode the header. Should not happen, so if it does,
			// shut down the connection to signal that the connection is broken.
			log.Println("rpc: gob error encoding response:", err)
			c.Close()
		}
		return
	}
	if err = c.enc.Encode(body); err != nil {
		if c.encBuf.Flush() == nil {
			// Was a gob problem encoding the body but the header has been written.
			// Shut down the connection to signal that the connection is broken.
			log.Println("rpc: gob error encoding body:", err)
			c.Close()
		}
		return
	}
	return c.encBuf.Flush()
}

func (c *gobServerCodec) Close() error {
	if c.closed {
		// Only call c.rwc.Close once; otherwise the semantics are undefined.
		return nil
	}
	c.closed = true
	return c.rwc.Close()
}

// ServeConn runs the server on a single connection.
// ServeConn blocks, serving the connection until the client hangs up.
// The caller typically invokes ServeConn in a go statement.
// ServeConn uses the gob wire format (see package gob) on the
// connection. To use an alternate codec, use ServeCodec.
// See NewClient's comment for information about concurrent access.
func (server *Server) ServeConn(conn io.ReadWriteCloser) {
	buf := bufio.NewWriter(conn)
	srv := &gobServerCodec{
		rwc:    conn,
		dec:    gob.NewDecoder(conn),
		enc:    gob.NewEncoder(buf),
		encBuf: buf,
	}
	server.ServeCodec(srv)
}

func (server *Server) ServeConnOnlyRecv(conn io.ReadWriteCloser) {
	srv := &gobServerCodec{
		rwc: conn,
		dec: gob.NewDecoder(conn),
	}
	if server.isOnlyRecvCache {
		srv.onlyReceiveChan = make(chan *receive_data, RECEIVE_LIST_DEFAULT_LENGTH)
		go server.ServeCodecOnlyRecvLoop(srv)
		server.ServeCodecOnlyRecvHandleData(srv)
	} else {
		server.ServeCodecOnlyRecv(srv)
	}
}

// ServeCodec is like ServeConn but uses the specified codec to
// decode requests and encode responses.
func (server *Server) ServeCodec(codec ServerCodec) {
	sending := new(sync.Mutex)
	wg := new(sync.WaitGroup)
	for {
		service, mtype, req, argv, replyv, keepReading, err := server.readRequest(codec)
		if err != nil {
			if debugLog && err != io.EOF {
				log.Println("rpc:", err)
			}
			if !keepReading {
				break
			}
			// send a response if we actually managed to read a header.
			if req != nil {
				server.sendResponse(sending, req, invalidRequest, codec, err.Error())
				server.freeRequest(req)
			}
			continue
		}
		wg.Add(1)
		go service.call(server, sending, wg, mtype, req, argv, replyv, codec)
	}
	// We've seen that there are no more requests.
	// Wait for responses to be sent before closing codec.
	wg.Wait()
	codec.Close()
}

func (server *Server) ServeCodecOnlyRecvLoop(codec *gobServerCodec) {
	defer func() {
		if err := recover(); err != nil {
			OutputCriticalStack(ServerLogErr, err)
			debug.PrintStack()
		}
	}()

	for {
		service, mtype, req, argv, replyv, keepReading, err := server.readRequest(codec)
		if err != nil {
			if debugLog && err != io.EOF {
				log.Println("rpc:", err)
			}
			if !keepReading {
				break
			}
			// send a response if we actually managed to read a header.
			if req != nil {
				server.freeRequest(req)
			}
			continue
		}
		codec.onlyReceiveChan <- &receive_data{
			srv:    service,
			mtype:  mtype,
			req:    req,
			argv:   argv,
			replyv: replyv,
		}
	}
}

func (server *Server) ServeCodecOnlyRecvHandleData(codec *gobServerCodec) {
	for {
		select {
		case d, ok := <-codec.onlyReceiveChan:
			if !ok {
				return
			}
			if d != nil {
				d.srv.call_only_recv(server, d.mtype, d.req, d.argv, d.replyv, codec)
			}
		}
	}
	codec.Close()
}

func (server *Server) ServeCodecOnlyRecv(codec ServerCodec) {
	for {
		service, mtype, req, argv, replyv, keepReading, err := server.readRequest(codec)
		if err != nil {
			if debugLog && err != io.EOF {
				log.Println("rpc:", err)
			}
			if !keepReading {
				break
			}
			// send a response if we actually managed to read a header.
			if req != nil {
				server.freeRequest(req)
			}
			continue
		}
		service.call_only_recv(server, mtype, req, argv, replyv, codec)
	}
	// We've seen that there are no more requests.
	// Wait for responses to be sent before closing codec.
	codec.Close()
}

// ServeRequest is like ServeCodec but synchronously serves a single request.
// It does not close the codec upon completion.
func (server *Server) ServeRequest(codec ServerCodec) error {
	sending := new(sync.Mutex)
	service, mtype, req, argv, replyv, keepReading, err := server.readRequest(codec)
	if err != nil {
		if !keepReading {
			return err
		}
		// send a response if we actually managed to read a header.
		if req != nil {
			server.sendResponse(sending, req, invalidRequest, codec, err.Error())
			server.freeRequest(req)
		}
		return err
	}
	service.call(server, sending, nil, mtype, req, argv, replyv, codec)
	return nil
}

func (server *Server) getRequest() *Request {
	server.reqLock.Lock()
	req := server.freeReq
	if req == nil {
		req = new(Request)
	} else {
		server.freeReq = req.next
		*req = Request{}
	}
	server.reqLock.Unlock()
	return req
}

func (server *Server) freeRequest(req *Request) {
	server.reqLock.Lock()
	req.next = server.freeReq
	server.freeReq = req
	server.reqLock.Unlock()
}

func (server *Server) getResponse() *Response {
	server.respLock.Lock()
	resp := server.freeResp
	if resp == nil {
		resp = new(Response)
	} else {
		server.freeResp = resp.next
		*resp = Response{}
	}
	server.respLock.Unlock()
	return resp
}

func (server *Server) freeResponse(resp *Response) {
	server.respLock.Lock()
	resp.next = server.freeResp
	server.freeResp = resp
	server.respLock.Unlock()
}

func (server *Server) readRequest(codec ServerCodec) (service *service, mtype *methodType, req *Request, argv, replyv reflect.Value, keepReading bool, err error) {
	service, mtype, req, keepReading, err = server.readRequestHeader(codec)
	if err != nil {
		if !keepReading {
			return
		}
		// discard body
		codec.ReadRequestBody(nil)
		return
	}

	// Decode the argument value.
	argIsValue := false // if true, need to indirect before calling.
	if mtype.ArgType.Kind() == reflect.Ptr {
		argv = reflect.New(mtype.ArgType.Elem())
	} else {
		argv = reflect.New(mtype.ArgType)
		argIsValue = true
	}
	// argv guaranteed to be a pointer now.
	if err = codec.ReadRequestBody(argv.Interface()); err != nil {
		return
	}
	if argIsValue {
		argv = argv.Elem()
	}

	replyv = reflect.New(mtype.ReplyType.Elem())

	switch mtype.ReplyType.Elem().Kind() {
	case reflect.Map:
		replyv.Elem().Set(reflect.MakeMap(mtype.ReplyType.Elem()))
	case reflect.Slice:
		replyv.Elem().Set(reflect.MakeSlice(mtype.ReplyType.Elem(), 0, 0))
	}
	return
}

func (server *Server) readRequestHeader(codec ServerCodec) (svc *service, mtype *methodType, req *Request, keepReading bool, err error) {
	// Grab the request header.
	req = server.getRequest()
	err = codec.ReadRequestHeader(req)
	if err != nil {
		req = nil
		if err == io.EOF || err == io.ErrUnexpectedEOF {
			return
		}
		err = errors.New("rpc: server cannot decode request: " + err.Error())
		return
	}

	// We read the header successfully. If we see an error now,
	// we can still recover and move on to the next request.
	keepReading = true

	dot := strings.LastIndex(req.ServiceMethod, ".")
	if dot < 0 {
		err = errors.New("rpc: service/method request ill-formed: " + req.ServiceMethod)
		return
	}
	serviceName := req.ServiceMethod[:dot]
	methodName := req.ServiceMethod[dot+1:]

	// Look up the request.
	svci, ok := server.serviceMap.Load(serviceName)
	if !ok {
		err = errors.New("rpc: can't find service " + req.ServiceMethod)
		return
	}
	svc = svci.(*service)
	mtype = svc.method[methodName]
	if mtype == nil {
		err = errors.New("rpc: can't find method " + req.ServiceMethod)
	}
	return
}

// Accept accepts connections on the listener and serves requests
// for each incoming connection. Accept blocks until the listener
// returns a non-nil error. The caller typically invokes Accept in a
// go statement.
func (server *Server) Accept(lis net.Listener) {
	var buf []byte = make([]byte, 1)
	for {
		conn, err := lis.Accept()
		if err != nil {
			log.Print("rpc.Serve: accept:", err.Error())
			return
		}
		// 读取头一个字节，判断连接类型
		if _, err = conn.Read(buf); err != nil {
			log.Print("rpc.Serve: connect type read err:", err.Error())
			continue
		}
		typ := int(buf[0])
		if typ == CONNECTION_TYPE_ONLY_READ {
			go server.ServeConn(conn)
		} else if typ == CONNECTION_TYPE_WRITE {
			go server.ServeConnOnlyRecv(conn)
		} else {
			log.Print("rpc.Serve: connection invalid type:", typ)
		}
	}
}

// Register publishes the receiver's methods in the DefaultServer.
func Register(rcvr any) error { return DefaultServer.Register(rcvr) }

// RegisterName is like Register but uses the provided name for the type
// instead of the receiver's concrete type.
func RegisterName(name string, rcvr any) error {
	return DefaultServer.RegisterName(name, rcvr)
}

// A ServerCodec implements reading of RPC requests and writing of
// RPC responses for the server side of an RPC session.
// The server calls ReadRequestHeader and ReadRequestBody in pairs
// to read requests from the connection, and it calls WriteResponse to
// write a response back. The server calls Close when finished with the
// connection. ReadRequestBody may be called with a nil
// argument to force the body of the request to be read and discarded.
// See NewClient's comment for information about concurrent access.
type ServerCodec interface {
	ReadRequestHeader(*Request) error
	ReadRequestBody(any) error
	WriteResponse(*Response, any) error

	// Close can be called multiple times and must be idempotent.
	Close() error
}

// ServeConn runs the DefaultServer on a single connection.
// ServeConn blocks, serving the connection until the client hangs up.
// The caller typically invokes ServeConn in a go statement.
// ServeConn uses the gob wire format (see package gob) on the
// connection. To use an alternate codec, use ServeCodec.
// See NewClient's comment for information about concurrent access.
func ServeConn(conn io.ReadWriteCloser) {
	DefaultServer.ServeConn(conn)
}

// ServeCodec is like ServeConn but uses the specified codec to
// decode requests and encode responses.
func ServeCodec(codec ServerCodec) {
	DefaultServer.ServeCodec(codec)
}

// ServeRequest is like ServeCodec but synchronously serves a single request.
// It does not close the codec upon completion.
func ServeRequest(codec ServerCodec) error {
	return DefaultServer.ServeRequest(codec)
}

// Accept accepts connections on the listener and serves requests
// to DefaultServer for each incoming connection.
// Accept blocks; the caller typically invokes it in a go statement.
func Accept(lis net.Listener) { DefaultServer.Accept(lis) }

`
