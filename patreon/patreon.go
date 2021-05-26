package patreon

type TierArray struct {
}

type PatreonSession struct {
}

func GetCampaignTiers(p *PatreonSession) (TierArray, error) {
	return TierArray{}, nil
}
