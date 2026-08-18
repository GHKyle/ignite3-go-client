package ignite3

import (
	"bytes"
	"fmt"

	"github.com/GHKyle/ignite3-go-client/binary/errors"
)


func (c *client) GetTables() (map[int64]string, error) {
	// request and response
	req := NewRequestOperation(OpTablesGet, c.RequestId, nil)
	res := NewResponseOperation(req.RequestId)
	
	// execute operation
	if err := c.Do(req, res); err != nil {
		return nil, errors.Wrapf(err, "failed to execute GET_TABLES operation")
	}
	if err := res.CheckStatus(); err != nil {
		return nil, err
	}

	// count of tables in result
	count, err := ReadPackedInt32(res.message)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to read pairs count")
	}

	data := make(map[int64]string)
	for i := 0; i < int(count); i++ {
		key, value, err := ReadPackedInt64String(res.message)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to read table infos with index %d", i)
		}
		data[key] = value
	}
	
	return data, nil
}

func (c *client) GetTable(table string) error {
	var bufr bytes.Buffer
	WritePackedString(&bufr, table)
	// request and response
	req := NewRequestOperation(OpTableGet, c.RequestId, bufr.Bytes())
	res := NewResponseOperation(req.RequestId)

	// execute operation
	if err := c.Do(req, res); err != nil {
		return  errors.Wrapf(err, "failed to execute GET_TABLE operation")
	}
	
	if err := res.CheckStatus(); err != nil {
		return err
	}
	
	// count of tables in result
	count, err := ReadPackedInt32(res.message)
	if err != nil {
		return errors.Wrapf(err, "failed to read pairs count")
	}
	// REMOVE ME
	fmt.Printf("DEBUG in (c *client) GetTables() : count = %x\n", count)
	
	data := make(map[int64]string)
	for i := 0; i < int(count); i++ {
		key, value, err := ReadPackedInt64String(res.message)
		if err != nil {
			return errors.Wrapf(err, "failed to read table infos with index %d", i)
		}
		data[key] = value
	}
	
	return nil
}
