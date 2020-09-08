import { oops, Mutable, showErrorMessage } from './util/util';

import { Observable } from './util/mixins';

import { Course, QueueApplication } from './QueueApplication';
import { OrderedQueue } from './OrderedQueue';
import { AppointmentsQueue } from './AppointmentsQueue';
import $ from 'jquery';

class Announcement {
    public readonly id: number;
    public readonly content: string;
    public readonly ts: string;

    public readonly queue: Page;

    private readonly elem: JQuery;

    constructor(data: { [index: string]: any }, queue: Page, elem: JQuery) {
        this.id = data['id'];
        this.content = data['content'];
        this.ts = new Date().toISOString();
        this.queue = queue;
        this.elem = elem;

        let panelBody: JQuery;
        this.elem.addClass('panel panel-info').append(
            (panelBody = $('<div class="panel-body bg-info"></div>')
                .append('<span class="glyphicon glyphicon-bullhorn"></span> ')
                .append($('<strong>' + this.content + '</strong>'))),
        );
        $('<button type="button" class="close adminOnly">&times;</button>')
            .appendTo(panelBody)
            .click((e) => {
                // TODO: Remove ugly confirm
                if (
                    confirm(
                        'Are you sure you want to remove this announcement? This will remove the announcement for all students (it\'s not just client-side).\n\n' +
                            this.content,
                    )
                ) {
                    this.remove();
                }
            });
    }

    public remove() {
        $.ajax({
            type: 'DELETE',
            url:
                'api/queues/' +
                this.queue.queueId +
                '/announcements/' +
                this.id,
            success: () => {
                this.queue.refresh();
            },
            error: oops,
        });
    }
}

export type QueueKind = 'ordered' | 'appointments';

function createQueue(
    data: { [index: string]: any },
    page: Page,
    queueKind: QueueKind,
    elem: JQuery,
) {
    if (queueKind === 'ordered') {
        return new OrderedQueue(data, page, elem);
    } else {
        return new AppointmentsQueue(data, page, elem);
    }
}

export class Page {
    private static _name: 'Queue';

    public readonly observable = new Observable(this);

    public readonly course: Course;
    public readonly queue: OrderedQueue | AppointmentsQueue;

    public readonly queueId: string;
    public readonly location: string;
    public readonly name: string;
    public readonly mapImageSrc: string = '';

    public readonly isAdmin: boolean = false;
    public readonly lastRefresh: Date = new Date();

    public readonly refreshDisabled: boolean = false;
    private currentRefreshIndex = 0;

    protected readonly elem: JQuery;
    private readonly numEntriesElem: JQuery;
    private readonly lastRefreshElem: JQuery;
    private readonly statusMessageElem: JQuery;
    private readonly announcementContainerElem: JQuery;
    private readonly adminStatusElem: JQuery;

    private readonly queueElem: JQuery;

    constructor(
        data: { [index: string]: any },
        course: Course,
        queueKind: QueueKind,
        elem: JQuery,
    ) {
        this.course = course;

        this.queueId = data['id'];
        this.location = data['location'];
        this.name = data['name'];
        this.mapImageSrc = data['map'] ? data['map'] : '';
        this.elem = elem;

        this.isAdmin = false;
        this.currentRefreshIndex = 0;
        this.lastRefresh = new Date();
        this.refreshDisabled = false;

        this.announcementContainerElem = $('<div></div>').appendTo(this.elem);

        var statusElem = $('<p></p>').appendTo(this.elem);
        statusElem.append(
            $(
                '<span data-toggle="tooltip" title="Number of Students"><span class="glyphicon glyphicon-education"></span></span>',
            )
                .append(' ')
                .append((this.numEntriesElem = $('<span></span>'))),
        );
        statusElem.append('&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;');
        statusElem.append(
            $(
                '<span data-toggle="tooltip" title="Last Refresh"><span class="glyphicon glyphicon-refresh"></span></span>',
            )
                .append(' ')
                .append((this.lastRefreshElem = $('<span></span>'))),
        );
        statusElem.append('&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;');

        this.statusMessageElem = $('<span>Loading queue information...</span>');
        statusElem.append(this.statusMessageElem);

        this.adminStatusElem = $(
            '<span class="adminOnly"><b>&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;You are an admin for this queue.</b></span>',
        );
        statusElem.append(this.adminStatusElem);

        this.elem.find('[data-toggle="tooltip"]').tooltip();

        this.queueElem = $('<div></div>').appendTo(this.elem);
        this.queue = createQueue(data, this, queueKind, this.queueElem);

        this.userSignedIn(); // TODO change name to updateUser?
    }

    public setStatusMessage(message: string) {
        this.statusMessageElem.html(message);
    }

    public setNumEntries(num: number) {
        this.numEntriesElem.html('' + num);
    }

    public refreshResponse(data: { [index: string]: any }) {
        // Message for individual user
        if (data['message']) {
            QueueApplication.instance.message(data['message']);
        }

        // Announcement for this queue as a whole
        this.announcementContainerElem.empty();
        let announcementsData = <any[]>data['announcements'] || [];
        announcementsData.forEach((aData: any) => {
            let announcementElem = $('<div></div>').appendTo(
                this.announcementContainerElem,
            );
            new Announcement(aData, this, announcementElem);
        });
        if (announcementsData.length > 0) {
            this.announcementContainerElem.show();
        } else {
            this.announcementContainerElem.hide();
        }

        (<Date>this.lastRefresh) = new Date();
        this.lastRefreshElem.html(this.lastRefresh.toLocaleTimeString());
    }

    public makeActiveOnClick(elem: JQuery) {
        elem.click(() => {
            this.makeActive();
        });
    }

    public makeActive() {
        QueueApplication.instance.setActiveQueue(this);
        this.refresh();
    }

    public refreshRequest() {
        return $.ajax({
            type: 'GET',
            url: 'api/queues/' + this.queueId,
            dataType: 'json',
        });
    }

    public refresh() {
        // myRefreshIndex is captured in a closure with the callback.
        // if refresh had been called again, the index won't match and
        // we don't do anything. this prevents the situation where someone
        // signs up but then a pending request from before they did so finishes
        // and causes it to look like they were immediately removed. this also
        // fixes a similar problem when an admin removes someone but then a
        // pending refresh makes them pop back up temporarily.
        this.currentRefreshIndex += 1;
        var myRefreshIndex = this.currentRefreshIndex;

        Promise.all([this.refreshRequest(), this.queue.refreshRequest()])
            .then((results) => {
                let myData = results[0];
                let queueData = results[1];
                // if another refresh has been requested, ignore the results of this one
                if (myRefreshIndex === this.currentRefreshIndex) {
                    if (!this.refreshDisabled) {
                        this.refreshResponse(myData);
                        this.queue.refreshResponse(queueData);
                    }
                }
            })
            .catch(oops);

        // return $.ajax({
        //     type: "POST",
        //     url: "api/list",
        //     data: {
        //         queueId: this.queueId
        //     },
        //     dataType: "json",
        //     success: (data) => {
        //         // if another refresh has been requested, ignore the results of this one
        //         if (myRefreshIndex === this.currentRefreshIndex){
        //             if (!this.refreshDisabled) {
        //                 this.refreshResponse(data);
        //             }
        //         }
        //     },
        //     error: oops
        // });
    }

    public cancelIncomingRefresh() {
        this.currentRefreshIndex += 1;
    }

    public disableRefresh() {
        (<Mutable<this>>this).refreshDisabled = true;
    }

    public enableRefresh() {
        (<Mutable<this>>this).refreshDisabled = false;
    }

    public setAdmin(isAdmin: boolean) {
        var oldAdmin = this.isAdmin;
        (<boolean>this.isAdmin) = isAdmin;

        // If our privileges change, hit the server for appropriate data,
        // because it gives out different things for normal vs. admin
        if (oldAdmin != this.isAdmin) {
            this.refresh();
        }
    }

    private userSignedIn() {
        this.observable.send('userSignedIn');
    }

    public hasMap() {
        return this.mapImageSrc !== '';
    }

    public updateGroups(data: any) {
        $.ajax({
            type: 'PUT',
            url: `api/queues/${this.queueId}/groups`,
            contentType: 'application/json',
            data: data,
            success: function (data) {
                alert('groups uploaded successfully');
            },
            error: oops,
        });
    }

    public updateConfiguration(options: { [index: string]: boolean }) {
        return $.ajax({
            type: 'PUT',
            url: 'api/queues/' + this.queueId + '/configuration',
            data: JSON.stringify(options),
            contentType: 'application/json',
            success: (data) => {},
            error: oops,
        });
    }

    public addAnnouncement(content: string) {
        return $.ajax({
            type: 'POST',
            url: 'api/queues/' + this.queueId + '/announcements',
            data: JSON.stringify({
                content: content,
            }),
            contentType: 'application/json',
            success: () => {
                this.refresh();
            },
            error: oops,
        });
    }
}
