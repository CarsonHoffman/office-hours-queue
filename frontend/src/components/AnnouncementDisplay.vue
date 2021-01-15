<template>
	<div class="notification is-warning">
		<button class="delete" v-if="admin" @click="removeAnnouncement"></button>
		<p v-html="linkifiedContent" class="announcement"></p>
	</div>
</template>

<script lang="ts">
import Vue from 'vue';
import { Component, Prop } from 'vue-property-decorator';
import linkifyStr from 'linkifyjs/string';

import Announcement from '../types/Announcement';
import Queue from '../types/Queue';
import ErrorDialog from '../util/ErrorDialog';

@Component
export default class AnnouncementDisplay extends Vue {
	@Prop({ required: true }) readonly announcement!: Announcement;
	@Prop({ required: true }) readonly queue!: Queue;
	@Prop({ required: true }) admin!: boolean;

	get linkifiedContent() {
		return linkifyStr(this.announcement.content, {
			defaultProtocol: 'https',
		});
	}

	removeAnnouncement() {
		this.$buefy.dialog.confirm({
			title: 'Delete Announcement',
			message: `Are you sure you want to delete this announcement? <b>There's no undo; this will remove this announcement for everyone.</b>`,
			type: 'is-danger',
			hasIcon: true,
			onConfirm: () => {
				fetch(
					process.env.BASE_URL +
						`api/queues/${this.queue.id}/announcements/${this.announcement.id}`,
					{
						method: 'DELETE',
					}
				).then((res) => {
					if (res.status !== 204) {
						return ErrorDialog(res);
					}
				});
			},
		});
	}
}
</script>

<style scoped>
.announcement {
	overflow-wrap: break-word;
}
</style>
