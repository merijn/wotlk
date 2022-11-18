package warrior

import (
	"strconv"
	"time"

	"github.com/wowsims/wotlk/sim/core"
)

var DeepWoundsActionID = core.ActionID{SpellID: 12867}

func (warrior *Warrior) applyDeepWounds() {
	if warrior.Talents.DeepWounds == 0 {
		return
	}

	warrior.DeepWounds = warrior.RegisterSpell(core.SpellConfig{
		ActionID:    DeepWoundsActionID,
		SpellSchool: core.SpellSchoolPhysical,
		ProcMask:    core.ProcMaskEmpty,
		Flags:       core.SpellFlagNoOnCastComplete | core.SpellFlagIgnoreModifiers,

		DamageMultiplier: 1,
		ThreatMultiplier: 1,

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			warrior.DeepWoundsDots[target.Index].ApplyOrReset(sim)
			spell.CalcAndDealOutcome(sim, target, spell.OutcomeAlwaysHit)
		},
	})

	warrior.RegisterAura(core.Aura{
		Label:    "Deep Wounds Talent",
		Duration: core.NeverExpires,
		OnReset: func(aura *core.Aura, sim *core.Simulation) {
			aura.Activate(sim)
		},
		OnSpellHitDealt: func(aura *core.Aura, sim *core.Simulation, spell *core.Spell, result *core.SpellResult) {
			if spell.ProcMask.Matches(core.ProcMaskEmpty) || !spell.SpellSchool.Matches(core.SpellSchoolPhysical) {
				return
			}
			if result.Outcome.Matches(core.OutcomeCrit) {
				warrior.procDeepWounds(sim, result.Target, spell.IsMH())
				warrior.procBloodFrenzy(sim, result, time.Second*6)
			}
		},
	})
}

func (warrior *Warrior) newDeepWoundsDot(target *core.Unit) *core.Dot {
	return core.NewDot(core.Dot{
		Spell: warrior.DeepWounds,
		Aura: target.RegisterAura(core.Aura{
			Label:    "DeepWounds-" + strconv.Itoa(int(warrior.Index)),
			ActionID: DeepWoundsActionID,
		}),
		NumberOfTicks: 6,
		TickLength:    time.Second * 1,

		OnTick: func(sim *core.Simulation, target *core.Unit, dot *core.Dot) {
			dot.SnapshotAttackerMultiplier = target.PseudoStats.PeriodicPhysicalDamageTakenMultiplier
			dot.CalcAndDealPeriodicSnapshotDamage(sim, target, dot.OutcomeTick)
		},
		SnapshotAttackerMultiplier: 1,
	})
}

func (warrior *Warrior) procDeepWounds(sim *core.Simulation, target *core.Unit, isMh bool) {
	dot := warrior.DeepWoundsDots[target.Index]

	var outstandingDamage float64
	if dot.IsActive() {
		outstandingDamage = dot.SnapshotBaseDamage * float64(dot.NumberOfTicks-dot.TickCount)
	}

	attackTable := warrior.AttackTables[target.UnitIndex]
	var awd float64
	if isMh {
		adm := warrior.AutoAttacks.MHAuto.AttackerDamageMultiplier(attackTable)
		tdm := warrior.AutoAttacks.MHAuto.TargetDamageMultiplier(attackTable, false)
		awd = (warrior.AutoAttacks.MH.CalculateAverageWeaponDamage(dot.Spell.MeleeAttackPower()) + dot.Spell.BonusWeaponDamage()) * adm * tdm
	} else {
		adm := warrior.AutoAttacks.OHAuto.AttackerDamageMultiplier(attackTable)
		tdm := warrior.AutoAttacks.OHAuto.TargetDamageMultiplier(attackTable, false)
		awd = ((warrior.AutoAttacks.OH.CalculateAverageWeaponDamage(dot.Spell.MeleeAttackPower()) * 0.5) + dot.Spell.BonusWeaponDamage()) * adm * tdm
	}
	newDamage := awd * 0.16 * float64(warrior.Talents.DeepWounds)

	dot.SnapshotBaseDamage = (outstandingDamage + newDamage) / float64(dot.NumberOfTicks)
	warrior.DeepWounds.Cast(sim, target)
}
