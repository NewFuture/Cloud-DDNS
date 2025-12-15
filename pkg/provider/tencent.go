package provider

import (
	"fmt"
	"strings"

	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/profile"
	dnspod "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/dnspod/v20210323"
)

type TencentProvider struct {
	secretId  string
	secretKey string
}

func NewTencentProvider(id, key string) *TencentProvider {
	return &TencentProvider{secretId: id, secretKey: key}
}

func (p *TencentProvider) UpdateRecord(fullDomain string, ip string) error {
	// 拆分域名
	parts := strings.Split(fullDomain, ".")
	if len(parts) < 2 {
		return fmt.Errorf("invalid domain format: %s (expected at least 2 parts)", fullDomain)
	}
	domain := strings.Join(parts[len(parts)-2:], ".")
	subDomain := strings.Join(parts[:len(parts)-2], ".")
	if subDomain == "" {
		subDomain = "@"
	}

	// 初始化客户端
	credential := common.NewCredential(p.secretId, p.secretKey)
	cpf := profile.NewClientProfile()
	client, err := dnspod.NewClient(credential, "", cpf)
	if err != nil {
		return err
	}

	// 1. 查询现有记录
	describeReq := dnspod.NewDescribeRecordListRequest()
	describeReq.Domain = common.StringPtr(domain)
	describeReq.Subdomain = common.StringPtr(subDomain)
	describeReq.RecordType = common.StringPtr("A")

	describeResp, err := client.DescribeRecordList(describeReq)
	if err != nil {
		return err
	}

	// 2. 判断是否需要更新
	if describeResp.Response.RecordList != nil && len(describeResp.Response.RecordList) > 0 {
		record := describeResp.Response.RecordList[0]
		// Check if Value is not nil before dereferencing
		if record.Value != nil && *record.Value == ip {
			return nil // IP 相同，无需更新
		}

		// 更新记录
		modifyReq := dnspod.NewModifyRecordRequest()
		modifyReq.Domain = common.StringPtr(domain)
		modifyReq.RecordId = record.RecordId
		modifyReq.SubDomain = common.StringPtr(subDomain)
		modifyReq.RecordType = common.StringPtr("A")
		modifyReq.RecordLine = common.StringPtr("默认")
		modifyReq.Value = common.StringPtr(ip)
		_, err = client.ModifyRecord(modifyReq)
		return err
	}

	// 3. 添加新记录
	createReq := dnspod.NewCreateRecordRequest()
	createReq.Domain = common.StringPtr(domain)
	createReq.SubDomain = common.StringPtr(subDomain)
	createReq.RecordType = common.StringPtr("A")
	createReq.RecordLine = common.StringPtr("默认")
	createReq.Value = common.StringPtr(ip)
	_, err = client.CreateRecord(createReq)
	return err
}
