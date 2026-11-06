package helpers

import (
	"fmt"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
)

// MaxExactTimestampMs bounds the range within which the `last_run` attributes
// are guaranteed to keep every single millisecond.
//
// Those attributes are declared schema.TypeFloat (see the 32-bit safety note on
// the three schemas, #543), so the SDK stores them as a float64. A float64
// represents EVERY integer in [-2^53, +2^53]; beyond that range it represents
// only some of them — 9007199254740992 and 9007199254740994 survive, while
// 9007199254740993 is rounded to the former. The bound is therefore the end of
// the contiguous exactly-representable range, not a hard ceiling: it is
// deliberately conservative, because a check that accepted "this particular
// value happens to be representable" would let the neighbouring one through
// rounded. The API documents `lastRun.start` / `lastRun.end` as int64, a wider
// domain, so the bound is checked rather than assumed.
//
// 2^53 ms is year 287396, so no timestamp a replication run can legitimately
// produce comes close; the check exists because the doctrine forbids writing a
// tracked field the provider knows to be wrong, not because the API is expected
// to reach it.
const MaxExactTimestampMs int64 = 1 << 53

// ValidateOpenIaaSReplicationPolicyLastRun fails closed when a `lastRun`
// timestamp falls outside the range whose millisecond precision the state can
// guarantee.
//
// It is deliberately NOT part of FlattenOpenIaaSReplicationPolicy: the mapping
// layer must stay a pure, total projection of the client object (it is exercised
// as such by the generic flatten harnesses), and a decision that refuses a read
// belongs to the Read paths that can surface a diagnostic. Every Read that
// flattens a policy calls this first.
//
// It never clamps, zeroes, omits or drops anything: an out-of-range timestamp
// is reported with the offending value so the platform side can be fixed, and
// the state keeps whatever it already held.
func ValidateOpenIaaSReplicationPolicyLastRun(policy *client.OpenIaaSReplicationPolicy) error {
	if policy == nil {
		return nil
	}
	for _, f := range []struct {
		name  string
		value int64
	}{
		{"start", policy.LastRun.Start},
		{"end", policy.LastRun.End},
	} {
		if f.value > MaxExactTimestampMs || f.value < -MaxExactTimestampMs {
			return fmt.Errorf(
				"replication policy %q (%s): the API returned last_run.%s = %d, which is outside the range ±%d within which this provider guarantees "+
					"millisecond precision; the Terraform state stores the value as a 64-bit float, which no longer represents every millisecond beyond "+
					"that bound, so the recorded timestamp could differ from the platform. Refusing to record a value whose exactness cannot be "+
					"guaranteed — report the timestamp to the Cloud Temple support so the API answer can be corrected",
				policy.Name, policy.ID, f.name, f.value, MaxExactTimestampMs,
			)
		}
	}
	return nil
}

// FlattenOpenIaaSReplicationPolicy convertit un objet OpenIaaSReplicationPolicy
// en une map compatible avec le schéma Terraform.
//
// `start` / `end` sont émis tels quels, en int64 : le writer du SDK les convertit
// en float64 pour les attributs TypeFloat, conversion exacte dans la plage
// garantie par ValidateOpenIaaSReplicationPolicyLastRun, que les Read appellent
// avant de flatten (#543).
func FlattenOpenIaaSReplicationPolicy(policy *client.OpenIaaSReplicationPolicy) map[string]interface{} {
	return map[string]interface{}{
		"id":                    policy.ID,
		"name":                  policy.Name,
		"machine_manager_id":    policy.MachineManager.ID,
		"storage_repository_id": policy.StorageRepository.ID,
		"pool_id":               policy.Pool.ID,
		"interval": []interface{}{
			map[string]interface{}{
				"hours":   policy.Interval.Hours,
				"minutes": policy.Interval.Minutes,
			},
		},
		"last_run": []interface{}{
			map[string]interface{}{
				"start":  policy.LastRun.Start,
				"end":    policy.LastRun.End,
				"status": policy.LastRun.Status,
			},
		},
	}
}
