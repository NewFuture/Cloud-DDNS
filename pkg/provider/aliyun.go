package provider

import (
	"fmt"

	alidns "github.com/alibabacloud-go/alidns-20150109/v4/client"
	openapi "github.com/alibabacloud-go/darabonba-openapi/v2/client"
	"github.com/alibabacloud-go/tea/tea"
)

type AliyunProvider struct {
	accessKey string
	secretKey string
}

func NewAliyunProvider(ak, sk string) *AliyunProvider {
	return &AliyunProvider{accessKey: ak, secretKey: sk}
}

func (p *AliyunProvider) UpdateRecord(fullDomain string, ip string) error {
	// 使用统一的域名解析函数
	domainName, rr, err := ParseDomain(fullDomain)
	if err != nil {
		return fmt.Errorf("invalid domain format: %s (%v)", fullDomain, err)
	}

	// 初始化客户端
	config := &openapi.Config{
		AccessKeyId:     tea.String(p.accessKey),
		AccessKeySecret: tea.String(p.secretKey),
		Endpoint:        tea.String("alidns.aliyuncs.com"),
	}
	client, err := alidns.NewClient(config)
	if err != nil {
		return err
	}

	// 1. 查询现有记录
	searchReq := &alidns.DescribeDomainRecordsRequest{
		DomainName: tea.String(domainName),
		RRKeyWord:  tea.String(rr),
	}
	resp, err := client.DescribeDomainRecords(searchReq)
	if err != nil {
		return err
	}

	// 2. 判断是否需要更新
	var recordId *string
	for _, r := range resp.Body.DomainRecords.Record {
		// Check if RR is not nil before dereferencing
		if r.RR != nil && *r.RR == rr {
			// Check if Value is not nil before dereferencing
			if r.Value != nil && *r.Value == ip {
				return nil // IP 相同，无需更新
			}
			recordId = r.RecordId
			break
		}
	}

	// 3. 执行更新或添加
	if recordId != nil {
		_, err = client.UpdateDomainRecord(&alidns.UpdateDomainRecordRequest{
			RecordId: recordId,
			RR:       tea.String(rr),
			Type:     tea.String("A"),
			Value:    tea.String(ip),
		})
	} else {
		_, err = client.AddDomainRecord(&alidns.AddDomainRecordRequest{
			DomainName: tea.String(domainName),
			RR:         tea.String(rr),
			Type:       tea.String("A"),
			Value:      tea.String(ip),
		})
	}
	return err
}
