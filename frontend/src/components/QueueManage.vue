<template>
	<div class="modal-card" style="width: auto">
		<header class="modal-card-head">
			<p class="modal-card-title">Manage Queue</p>
			<button type="button" class="delete" @click="$emit('close')" />
		</header>
		<section class="modal-card-body">
			<div class="block">
				<button class="button is-primary" @click="addAnnouncement">
					Add Announcement
				</button>
			</div>
			<div class="block">
				<p class="title">Queue Settings</p>
				<div class="field">
					<b-checkbox v-model="configuration['prevent_unregistered']"
						>Prevent students not registered in any group from signing up at
						all</b-checkbox
					>
				</div>
				<div class="field">
					<b-checkbox v-model="configuration['prevent_groups']"
						>Prevent multiple students in a group from signing up at the same
						time</b-checkbox
					>
				</div>
				<div class="field" v-if="type === 'ordered'">
					<b-checkbox v-model="configuration['prioritize_new']"
						>Prioritize students who signed up for the first time this
						day</b-checkbox
					>
				</div>
				<div class="field" v-if="type === 'ordered'">
					<b-checkbox v-model="configuration['prevent_groups_boost']"
						>Prevent multiple students in a group from receiving the boost for
						first question per day</b-checkbox
					>
				</div>
				<button
					class="button is-primary"
					@click="$emit('configurationSaved', configuration)"
				>
					Save Queue Settings
				</button>
			</div>
			<div class="block">
				<p class="title">Queue Groups</p>
				<b-field
					label="Groups input (JSON; array of groups, each group is array of emails of members)"
				>
					<b-input
						type="textarea"
						:placeholder="JSON.stringify(groupsPlaceholder, null, 4)"
						rows="14"
						v-model="groupsInput"
					></b-input>
				</b-field>
				<div class="level">
					<div class="level-left">
						<div class="buttons level-item">
							<button class="button is-primary" @click="addGroups">
								Add Groups
							</button>
							<button class="button is-warning" @click="setGroups">
								Overwrite Groups
							</button>
						</div>
					</div>
					<div class="level-right">
						<button
							class="button is-success level-item"
							@click="$emit('groupsSaved', groups)"
						>
							Upload Groups
						</button>
					</div>
				</div>
				<p class="title is-4">Groups</p>
				<div
					class="box"
					v-for="(group, i) in groups"
					:key="i"
					style="position: relative"
				>
					<button
						class="button is-white is-small delete-button"
						@click="removeGroup(i)"
					>
						<span class="icon"><font-awesome-icon icon="times"/></span>
					</button>
					<div
						class="level icon-row is-mobile"
						v-for="email in group"
						:key="email"
					>
						<div class="level-left">
							<p class="level-item">{{ email }}</p>
							<button
								class="button is-white is-small level-item"
								@click="removeMember(i, email)"
							>
								<span class="icon"><font-awesome-icon icon="times"/></span>
							</button>
						</div>
					</div>
				</div>
			</div>
		</section>
		<footer class="modal-card-foot">
			<button class="button" type="button" @click="$emit('close')">
				Close
			</button>
		</footer>
	</div>
</template>

<script lang="ts">
import { Component, Prop, Vue } from 'vue-property-decorator';
import moment, { Moment } from 'moment-timezone';
import Queue from '@/types/Queue';

import { library } from '@fortawesome/fontawesome-svg-core';
import { faTimes } from '@fortawesome/free-solid-svg-icons';

library.add(faTimes);

@Component({})
export default class QueueManage extends Vue {
	@Prop({ required: true })
	type!: 'ordered' | 'appointments';

	@Prop({ required: true })
	defaultConfiguration!: { [index: string]: any };

	configuration: { [index: string]: any } = {};

	@Prop({ required: true })
	defaultGroups!: string[][];

	groups: string[][] = [];

	groupsInput = '';

	constructor() {
		super();
		this.configuration = { ...this.defaultConfiguration };
		for (let i = 0; i < this.defaultGroups.length; i++) {
			this.groups.push([...this.defaultGroups[i]]);
		}
	}

	addAnnouncement() {
		this.$buefy.dialog.prompt({
			message: 'Announcement content:',
			confirmText: 'Add Announcement',
			onConfirm: (content) => this.$emit('announcementAdded', content),
		});
	}

	groupsPlaceholder = [
		['member 1 of group 1', 'member 2 of group 1'],
		['member 1 of group 2', 'member 2 of group 2', 'member 3 of group 2'],
		['only member of group 3'],
	];

	validateGroupsArray(obj: any, currentGroups: string[][]): boolean {
		if (!Array.isArray(obj)) {
			this.$buefy.dialog.alert({
				message: 'Input does not contain array of arrays of strings.',
				type: 'is-danger',
			});
			return false;
		}
		const emailsSeen = new Set<string>();
		for (const g of currentGroups) {
			for (const e of g) {
				emailsSeen.add(e);
			}
		}
		for (const g of obj) {
			if (!Array.isArray(g)) {
				this.$buefy.dialog.alert({
					message: 'Input does not contain array of arrays of strings.',
					type: 'is-danger',
				});
				return false;
			}
			for (const e of g) {
				if (typeof e !== 'string') {
					this.$buefy.dialog.alert({
						message: 'Input contains a non-string email.',
						type: 'is-danger',
					});
					return false;
				}
				if (emailsSeen.has(e)) {
					this.$buefy.dialog.alert({
						message: `Email ${e} appears in more than one group!`,
						type: 'is-danger',
					});
					return false;
				}
				emailsSeen.add(e);
			}
		}

		return true;
	}

	addGroups() {
		try {
			const groups = JSON.parse(this.groupsInput);
			if (!this.validateGroupsArray(groups, this.groups)) {
				return;
			}
			for (const g of groups) {
				this.groups.push([...g]);
			}
		} catch {
			this.$buefy.dialog.alert({
				message: 'Input is not valid JSON.',
				type: 'is-danger',
			});
		}
	}

	setGroups() {
		try {
			const groups = JSON.parse(this.groupsInput);
			if (!this.validateGroupsArray(groups, [])) {
				return;
			}
			this.groups = [];
			for (const g of groups) {
				this.groups.push([...g]);
			}
		} catch {
			this.$buefy.dialog.alert({
				message: 'Input is not valid JSON.',
				type: 'is-danger',
			});
		}
	}

	removeGroup(i: number) {
		this.groups.splice(i, 1);
	}

	removeMember(i: number, email: string) {
		this.groups.splice(
			i,
			1,
			this.groups[i].filter((e) => e !== email)
		);

		if (this.groups[i].length === 0) {
			this.removeGroup(i);
		}
	}
}
</script>

<style scoped>
.delete-button {
	position: absolute;
	top: 5px;
	right: 5px;
}
</style>
