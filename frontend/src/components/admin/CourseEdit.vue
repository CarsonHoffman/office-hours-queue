<template>
	<div class="modal-card" style="width: auto">
		<header class="modal-card-head">
			<p class="modal-card-title">Course Info</p>
			<button type="button" class="delete" @click="$emit('close')" />
		</header>
		<section class="modal-card-body">
			<div class="block">
				<b-field label="Short Name">
					<b-input v-model="shortName" />
				</b-field>
				<b-field label="Full Name">
					<b-input v-model="fullName" />
				</b-field>
				<b-field label="Course Admins">
					<b-input type="textarea" v-model="adminsText" />
				</b-field>
			</div>
		</section>
		<footer class="modal-card-foot">
			<button class="button" type="button" @click="$emit('close')">
				Close
			</button>
			<button class="button is-success" type="button" @click="saveCourse">
				Save
			</button>
		</footer>
	</div>
</template>

<script lang="ts">
import { Component, Prop, Vue } from 'vue-property-decorator';
import moment, { Moment } from 'moment-timezone';
import Queue from '@/types/Queue';
import EscapeHTML from '@/util/Sanitization';

import { library } from '@fortawesome/fontawesome-svg-core';
import { faTimes } from '@fortawesome/free-solid-svg-icons';

library.add(faTimes);

@Component({})
export default class QueueManage extends Vue {
	@Prop({ required: true }) defaultShortName!: string;
	@Prop({ required: true }) defaultFullName!: string;
	@Prop({ required: true }) defaultAdmins!: string[];

	shortName = '';
	fullName = '';
	adminsText = '';

	created() {
		this.shortName = this.defaultShortName;
		this.fullName = this.defaultFullName;
		this.adminsText = JSON.stringify(this.defaultAdmins, null, 4);
	}

	saveCourse() {
		try {
			const admins: string[] = JSON.parse(this.adminsText);
			if (!Array.isArray(admins) || admins.some((a) => typeof a !== 'string')) {
				this.$buefy.dialog.alert({
					message: 'Admins input is not array of strings',
					type: 'is-danger',
				});
				return;
			}

			const allAdmins = new Set<string>();
			for (const a of admins) {
				if (allAdmins.has(a)) {
					this.$buefy.dialog.alert({
						message: `User ${EscapeHTML(
							a
						)} appears in the admins array more than once.`,
						type: 'is-danger',
					});
					return;
				}
				allAdmins.add(a);
			}

			this.$emit('saved', this.shortName, this.fullName, admins);
		} catch {
			this.$buefy.dialog.alert({
				message: 'Admins input is not valid JSON.',
				type: 'is-danger',
			});
		}
	}
}
</script>
