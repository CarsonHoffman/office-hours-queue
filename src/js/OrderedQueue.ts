import { Course, QueueApplication, User } from './QueueApplication';
import {
    MessageResponses,
    messageResponse,
    addListener,
    Observable,
    Message,
} from './util/mixins';
import { oops, showErrorMessage, Mutable, asMutable } from './util/util';
import { Page } from './queue';
import $ from 'jquery';
import moment, { max, Moment } from 'moment-timezone';

var ANIMATION_DELAY = 500;

export class OrderedQueue {
    public readonly kind = 'ordered';

    public readonly observable = new Observable(this);

    public readonly page: Page;

    public readonly myRequest: QueueEntry | null = null;

    public readonly isOpen: boolean = false;
    public readonly numEntries: number;

    private readonly elem: JQuery;
    private readonly adminControlsElem: JQuery;
    private readonly studentControlsElem: JQuery;
    private readonly queueElem: JQuery;
    private readonly stackElem: JQuery;

    private readonly adminControls: AdminControls;
    private readonly studentControls: StudentControls;

    constructor(data: { [index: string]: any }, page: Page, elem: JQuery) {
        this.page = page;
        this.elem = elem;

        this.isOpen = false;
        this.numEntries = 0;

        this.adminControlsElem = $(
            '<div class="panel panel-default adminOnly"><div class="panel-body"></div></div>',
        )
            .appendTo(this.elem)
            .find('.panel-body');

        this.adminControls = new AdminControls(this, this.adminControlsElem);

        this.studentControlsElem = $(
            '<div class="panel panel-default"><div class="panel-body"></div></div>',
        )
            .appendTo(this.elem)
            .find('.panel-body');

        this.studentControls = new StudentControls(
            this,
            this.studentControlsElem,
        );
        addListener(this, this.studentControls);
        addListener(this.page, this.studentControls);

        this.queueElem = $('<div></div>').appendTo(this.elem);
        this.stackElem = $('<div class="adminOnly"></div>').appendTo(this.elem);
    }

    // protected readonly refreshType = "POST";
    // protected refreshUrl() {
    //     return "api/list";
    // }
    // protected readonly refreshDataType = "json";
    // protected refreshData() {
    //     return {
    //         queueId: this.queueId
    //     }
    // }

    public refreshRequest() {
        return $.ajax({
            type: 'GET',
            url: 'api/queues/' + this.page.queueId,
            dataType: 'json',
        });
    }

    public refreshResponse(data: { [index: string]: any }) {
        (<boolean>this.isOpen) = data['open'];
        if (this.isOpen) {
            this.page.setStatusMessage('The queue is open.');
        } else {
            let schedule = data['schedule'];
            let halfHour = data['half_hour'];
            let nextOpen = -1;
            for (let i = halfHour; i < 48; ++i) {
                let scheduleType = schedule.charAt(i);
                if (scheduleType === 'o' || scheduleType === 'p') {
                    nextOpen = i;
                    break;
                }
            }

            if (nextOpen === -1) {
                this.page.setStatusMessage('The queue is closed for today.');
            } else {
                let d = new Date();
                d.setHours(0);
                d.setMinutes(0);
                d.setSeconds(0);

                let newDate = new Date(d.getTime() + nextOpen * 30 * 60000);
                this.page.setStatusMessage(
                    'The queue is closed right now. It will open at ' +
                        newDate.toLocaleTimeString() +
                        '.',
                );
            }
        }

        let queue = data['queue'];
        this.queueElem.empty();
        let queueEntries = [];
        let myRequest: QueueEntry | null = null;
        for (let i = 0; i < queue.length; ++i) {
            let item = queue[i];

            let itemElem = $("<li class='list-group-item'></li>");
            let entry = new QueueEntry(this, item, i, itemElem);
            queueEntries.push(entry);

            if (!myRequest && User.isMe(entry.email)) {
                myRequest = entry;
            }

            this.queueElem.append(itemElem);
        }
        this.setMyRequest(myRequest);

        this.observable.send('queueRefreshed');

        // console.log(JSON.stringify(data["stack"], null, 4));
        this.stackElem.html(
            '<h3>The Stack</h3><br /><p>Most recently removed at top</p><pre>' +
                JSON.stringify(data['stack'], null, 4) +
                '</pre>',
        );

        var oldNumEntries = this.numEntries;
        (<number>this.numEntries) = queue.length;
        if (this.page.isAdmin && oldNumEntries === 0 && this.numEntries > 0) {
            QueueApplication.instance.notify(
                'Request Received!',
                queueEntries[0].name,
            );
        }

        this.page.setNumEntries(this.numEntries);
    }

    public setMyRequest(myRequest: QueueEntry | null) {
        (<QueueEntry | null>this.myRequest) = myRequest;
        this.studentControls.myRequestSet();
    }

    public removeRequest(request: QueueEntry) {
        console.log(
            'attempting to remove ' +
                request.email +
                ' from queue ' +
                this.page.queueId,
        );
        this.page.disableRefresh();
        $.ajax({
            type: 'DELETE',
            url: 'api/queues/' + this.page.queueId + '/entries/' + request.id,
            success: () => {
                console.log(
                    'successfully removed ' +
                        request.email +
                        ' from queue ' +
                        this.page.queueId,
                );
                request.onRemove();
            },
            error: oops,
        }).always(() => {
            setTimeout(() => {
                this.page.enableRefresh();
                this.page.refresh();
            }, ANIMATION_DELAY);
        });
    }

    public clear() {
        return $.ajax({
            type: 'DELETE',
            url: 'api/queues/' + this.page.queueId + '/entries',
            success: () => {
                this.clearList();
            },
            error: oops,
        });
    }

    private clearList() {
        this.queueElem.children().slideUp();
    }

    public signUp(
        name: string,
        location: string,
        description: string,
        mapX: number = 0,
        mapY: number = 0,
    ) {
        return $.ajax({
            type: 'POST',
            url: 'api/queues/' + this.page.queueId + '/entries',
            data: JSON.stringify({
                name: name,
                location: location,
                mapX: mapX,
                mapY: mapY,
                description: description,
            }),
            contentType: 'application/json',
            dataType: 'json',
            success: (data) => {
                if (data['fail']) {
                    showErrorMessage(data['reason']);
                } else {
                    this.page.refresh();
                }
            },
            error: oops,
        });
    }

    public updateRequest(
        name: string,
        location: string,
        description: string,
        mapX?: number,
        mapY?: number,
    ) {
        return $.ajax({
            type: 'PUT',
            url:
                'api/queues/' +
                this.page.queueId +
                '/entries/' +
                this.myRequest!.id,
            data: JSON.stringify({
                name: name,
                location: location,
                mapX: mapX,
                mapY: mapY,
                description: description,
            }),
            contentType: 'application/json',
            success: (data) => {
                this.page.refresh();
            },
            error: oops,
        });
    }
}

class StudentControls {
    private static _name = 'StudentControls';

    private queue: OrderedQueue;

    private elem: JQuery;
    private signUpForm: SignUpForm;

    public readonly _act!: MessageResponses;

    constructor(queue: OrderedQueue, elem: JQuery) {
        this.queue = queue;
        this.elem = elem;

        let formElem = $('<div></div>').appendTo(this.elem);
        this.signUpForm = new SignUpForm(
            formElem,
            false,
            this.queue.page.mapImageSrc,
        );
        addListener(this.signUpForm, this);
    }

    public refreshSignUpEnabled() {
        var isEnabled =
            User.isUmich() && this.queue.isOpen && !this.queue.myRequest;
        this.signUpForm.setSignUpEnabled(isEnabled);
    }

    @messageResponse()
    private queueRefreshed() {
        this.refreshSignUpEnabled();
    }

    @messageResponse()
    private userSignedIn() {
        this.refreshSignUpEnabled();
    }

    public myRequestSet() {
        this.signUpForm.setMyRequest(this.queue.myRequest);
    }

    @messageResponse()
    private signUp(msg: Message<SignUpMessage>) {
        if (!this.queue.myRequest) {
            this.queue.signUp(
                msg.data.signUpName,
                msg.data.signUpLocation,
                msg.data.signUpDescription,
                msg.data.mapX,
                msg.data.mapY,
            );
        } else {
            this.queue.updateRequest(
                msg.data.signUpName,
                msg.data.signUpLocation,
                msg.data.signUpDescription,
                msg.data.mapX,
                msg.data.mapY,
            );
        }
    }

    @messageResponse()
    private removeRequest(msg: Message<QueueEntry>) {
        this.queue.removeRequest(msg.data);
    }
}

export interface SignUpMessage {
    readonly signUpName: string;
    readonly signUpLocation: string;
    readonly signUpDescription: string;
    readonly mapX: number;
    readonly mapY: number;
    readonly timeslot: number;
}

export interface Appointment {
    readonly kind: 'appointment';
    id: string;
    queueId: string;
    timeslot: number;
    duration: number;
    scheduledTime: Moment;
    studentEmail: string | undefined;
    staffEmail: string | undefined;
    name: string | undefined;
    location: string | undefined;
    description: string | undefined;
    mapX: number | undefined;
    mapY: number | undefined;
}

export interface SignUpAppointmentSlots {
    timeslot: number;
    duration: number;
    scheduledTime: Moment;
    numAvailable: number;
    numFilled: number;
}

export type AppointmentSchedule = SignUpAppointmentSlots[];

interface SignUpButtonContent {
    signUp: string;
    upToDate: string;
    update: string;
    remove: string;
}

const QUEUE_SIGN_UP_BUTTON_CONTENT: SignUpButtonContent = {
    signUp: 'Sign Up',
    upToDate: "<span class='glyphicon glyphicon-ok'></span> Request Updated",
    update: 'Update My Request',
    remove: 'Remove Me From Queue',
};

const APPOINTMENTS_SIGN_UP_BUTTON_CONTENT: SignUpButtonContent = {
    signUp: 'Schedule Appointment',
    upToDate: "<span class='glyphicon glyphicon-ok'></span> Up To Date",
    update: 'Update/Move My Appointment',
    remove: 'Cancel My Appointment',
};

export class SignUpForm<HasAppointments extends boolean = false> {
    public readonly hasMap: boolean;
    public readonly mapImgSrc?: string;

    public readonly myRequest:
        | (HasAppointments extends true ? Appointment : QueueEntry)
        | null = null;
    public readonly appointments!: HasAppointments extends true
        ? AppointmentSchedule
        : undefined;

    private formHasChanges: boolean;

    private elem: JQuery;
    private statusElem: JQuery;
    private signUpForm: JQuery;
    private signUpNameInput: JQuery;
    private signUpDescriptionInput: JQuery;
    private signUpLocationInput: JQuery;
    private signUpMap?: JQuery;
    private signUpPin?: JQuery;
    private mapX?: number;
    private mapY?: number;
    private appointmentsSlotsTable: JQuery;
    private appointmentHeaderElems?: readonly JQuery[];
    private appointmentHeaderElemsMap: { [index: number]: JQuery | undefined} = {};
    private selectedTimeslot?: number;
    private signUpButtons: JQuery;
    private updateRequestButtons: JQuery;
    private removeRequestButtons: JQuery;

    private signUpButtonContent: SignUpButtonContent;

    public readonly observable = new Observable(this);
    public readonly _act!: MessageResponses;

    private static _inst_id = 0;
    private _inst_id = SignUpForm._inst_id++;

    constructor(
        elem: JQuery,
        appointments: HasAppointments,
        mapImgSrc?: string,
    ) {
        this.elem = elem;
        this.hasMap = !!mapImgSrc;
        this.mapImgSrc = mapImgSrc;

        this.signUpButtonContent = appointments
            ? APPOINTMENTS_SIGN_UP_BUTTON_CONTENT
            : QUEUE_SIGN_UP_BUTTON_CONTENT;

        this.formHasChanges = false;

        let regularFormElem;
        this.appointmentsSlotsTable = $(
            '<table class="appointment-slots-table"></table>',
        );

        this.signUpForm = $(
            '<form id="signUpForm" role="form" class="form-horizontal"></form>',
        ).append(
            (regularFormElem = $('<div></div>')
                .append(
                    $('<div class="form-group"></div>')
                        .append(
                            '<label class="control-label col-sm-3" for="signUpName' +
                                this._inst_id +
                                '">Name:</label>',
                        )
                        .append(
                            $('<div class="col-sm-9"></div>').append(
                                (this.signUpNameInput = $(
                                    '<input type="text" class="form-control" id="signUpName' +
                                        this._inst_id +
                                        '" required="required" maxlength="30" placeholder="Nice to meet you!">',
                                )),
                            ),
                        ),
                )
                .append(
                    $('<div class="form-group"></div>')
                        .append(
                            '<label class="control-label col-sm-3" for="signUpDescription' +
                                this._inst_id +
                                '">Description:</label>',
                        )
                        .append(
                            $('<div class="col-sm-9"></div>').append(
                                (this.signUpDescriptionInput = $(
                                    '<input type="text" class="form-control" id="signUpDescription' +
                                        this._inst_id +
                                        '"required="required" maxlength="100" placeholder="e.g. Segfault in function X, using the map data structure, etc.">',
                                )),
                            ),
                        ),
                )
                .append(
                    $('<div class="form-group"></div>')
                        .append(
                            '<label class="control-label col-sm-3" for="signUpLocation' +
                                this._inst_id +
                                `">Meeting Link:</label>`,
                        )
                        .append(
                            $('<div class="col-sm-9"></div>').append(
                                (this.signUpLocationInput = $(
                                    '<input type="text" class="form-control" id="signUpLocation' +
                                        this._inst_id +
                                        '"required="required" maxlength="100" placeholder="">',
                                )),
                            ),
                        ),
                )
                .append(
                    !appointments
                        ? ''
                        : $('<div class="form-group"></div>')
                              .append(
                                  '<label class="control-label col-sm-3" for="signUpAppointmentSchedule' +
                                      this._inst_id +
                                      '">Appointments:</label>',
                              )
                              .append(
                                  $('<div class="col-sm-9"></div>').append(
                                      $(
                                          '<div style="overflow-x: scroll"></div>',
                                      ).append(this.appointmentsSlotsTable),
                                  ),
                              ),
                )
                .append(
                    '<div class="' +
                        (this.hasMap ? 'hidden-xs' : '') +
                        ` form-group"><div class="col-sm-offset-3 col-sm-9">
                    <button type="submit" class="btn btn-success queue-signUpButton">${this.signUpButtonContent.signUp}</button>
                    <button type="submit" class="btn btn-success queue-updateRequestButton" style="display:none;"></button>
					` +
                        (!appointments
                            ? ''
                            : '<button type="button" class="btn btn-danger queue-removeRequestButton" data-toggle="modal" data-target="#removeMyAppointmentDialog" style="display:none;"></button>') +
                        `</div></div>`,
                )),
        );

        this.elem.append(this.signUpForm);

        this.statusElem = $('<div></div>');
        this.elem.append(this.statusElem);

        this.signUpForm.find('input').on('input', () => {
            this.formChanged();
        });

        if (this.hasMap) {
            regularFormElem.addClass('col-xs-12 col-sm-8');
            regularFormElem.css('padding', '0');
            this.signUpForm.append(
                $(
                    '<div class="col-xs-12 col-sm-4" style="position: relative; padding:0"></div>',
                )
                    .append(
                        (this.signUpMap = $(
                            '<img src="img/' +
                                this.mapImgSrc +
                                '" class="queue-signUpMap" style="width:100%"></img>',
                        )),
                    )
                    .append(
                        (this.signUpPin = $(
                            '<span class="queue-locatePin"><span class="glyphicon glyphicon-map-marker" style="position:absolute; left:-1.3ch;top:-0.95em;"></span></span>',
                        )),
                    ),
            );

            // Add different layout for sign up button on small screens
            this.signUpForm.append(
                $(`<div class="visible-xs col-xs-12" style="padding: 0;"><div class="form-group"><div class="col-sm-offset-3 col-sm-9">
                <button type="submit" class="btn btn-success queue-signUpButton">${this.signUpButtonContent.signUp}</button> 
                <button type="submit" class="btn btn-success queue-updateRequestButton" style="display:none;"></button>
                <button type="button" class="btn btn-success queue-removeRequestButton" data-toggle="modal" data-target="#removeMyAppointmentDialog" style="display:none;"></button>
                </div></div></div>`),
            );

            var pin = this.signUpPin;
            this.mapX = 50;
            this.mapY = 50;
            let self = this;
            this.signUpMap.click(function (e) {
                //Offset mouse Position
                // Use ! for non-null assertion
                self.mapX =
                    (100 * Math.trunc(e.pageX - $(this).offset()!.left)) /
                    $(this).width()!;
                self.mapY =
                    (100 * Math.trunc(e.pageY - $(this).offset()!.top)) /
                    $(this).height()!;
                // var pinLeft = mapX - pin.width()/2;
                // var pinTop = mapY - pin.height();
                pin.css('left', self.mapX + '%');
                pin.css('top', self.mapY + '%');
                self.formChanged();
                //            alert("x:" + mapX + ", y:" + mapY);
            });

            // Disable regular location input
            this.signUpLocationInput.val('Click on the map!');
            this.signUpLocationInput.prop('disabled', true);
        }

        this.signUpForm.submit((e) => {
            e.preventDefault();
            let signUpName: string = <string>this.signUpNameInput.val();
            let signUpDescription: string = <string>(
                this.signUpDescriptionInput.val()
            );
            let signUpLocation: string = <string>this.signUpLocationInput.val();
            let signUpTimeslot = this.selectedTimeslot;

            if (
                !signUpName ||
                signUpName.length == 0 ||
                !signUpLocation ||
                signUpLocation.length == 0 ||
                !signUpDescription ||
                signUpDescription.length == 0
            ) {
                showErrorMessage('You must fill in all the fields.');
                return false;
            }

            if (this.hasAppointments() && !signUpTimeslot) {
                showErrorMessage('You must select an appointment timeslot.');
                return false;
            }

            let msg: SignUpMessage = {
                signUpName: signUpName,
                signUpLocation: signUpLocation,
                signUpDescription: signUpDescription,
                mapX: this.mapX ?? 0,
                mapY: this.mapY ?? 0,
                timeslot: signUpTimeslot ?? 0,
            };
            this.observable.send('signUp', msg);

            this.formHasChanges = false;
            this.updateRequestButtons.removeClass('btn-warning');
            this.updateRequestButtons.addClass('btn-success');
            this.updateRequestButtons.prop('disabled', true);
            this.updateRequestButtons.html(this.signUpButtonContent.upToDate);
            return false;
        });

        this.signUpButtons = this.signUpForm.find('button.queue-signUpButton');
        this.updateRequestButtons = this.signUpForm
            .find('button.queue-updateRequestButton')
            .prop('disabled', true)
            .html(this.signUpButtonContent.upToDate);

        this.removeRequestButtons = this.signUpForm
            .find('button.queue-removeRequestButton')
            .html(this.signUpButtonContent.remove);
        // .click(() => this.myRequest && this.observable.send("removeRequest", this.myRequest));
    }

    public hasAppointments(): this is SignUpForm<true> {
        return !!this.appointments;
    }

    public setMyRequest(
        this: SignUpForm<false>,
        request: QueueEntry | null,
    ): void;
    public setMyRequest(
        this: SignUpForm<true>,
        request: Appointment | null,
    ): void;
    public setMyRequest(request: QueueEntry | Appointment | null) {
        asMutable(this).myRequest = <any>request;

        this.statusElem.html('');
        if (request) {
            if (!this.formHasChanges) {
                this.signUpNameInput.val(request.name || '');
                this.signUpDescriptionInput.val(request.description || '');
                this.signUpLocationInput.val(request.location || '');

                if (this.hasMap) {
                    this.mapX = request.mapX;
                    this.mapY = request.mapY;
                    this.signUpPin!.css('left', this.mapX + '%');
                    this.signUpPin!.css('top', this.mapY + '%');
                }

                if (request.kind === 'appointment') {
                    (<SignUpForm<true>>this).setSelectedTimeslot(
                        request.timeslot,
                    );
                }
            }

            if (request.kind === 'queue_entry') {
                this.statusElem.html(
                    'You are at position ' +
                        (request.index + 1) +
                        ' in the queue.',
                );
                if (request.tag) {
                    this.statusElem.prepend(
                        '<span class="label label-info">' +
                            request.tag +
                            '</span> ',
                    );
                }
            } else {
                // appointment
                this.statusElem.html(
                    'Your appointment is scheduled at ' +
                        request.scheduledTime.format('h:mma') +
                        '.',
                );
            }
        }
    }

    public setAppointments(
        this: SignUpForm<true>,
        appointments: AppointmentSchedule,
    ) {
        asMutable(this).appointments = appointments;

        // let buttonsElem = $("<div></div>").appendTo(this.appointmentsElem);

        let now = moment();

        // alert( start.toUTCString() + ':' + end.toUTCString() );
        // let table = $("<table></table>");
        this.appointmentsSlotsTable.html('');

        let maxAppts = appointments.reduce(
            (prev, current) => Math.max(prev, current.numAvailable),
            0,
        );

        // filter to only times with some appointments available,
        // or the first in a sequence of no availability, which will be rendered as a "gap"
        appointments = filterAppointmentsSchedule(appointments);

        // header row with times
        let headerRow = $('<tr></tr>').appendTo(this.appointmentsSlotsTable);
        let headerElems: JQuery[] = [];
        this.appointmentHeaderElemsMap = {};
        this.appointmentHeaderElems = headerElems;
        appointments.forEach((slots, i) => {
            let headerElem: JQuery;
            if (slots.numAvailable > 0) {
                headerElem = $(
                    `<th class="appointment-slots-header"><span>${slots.scheduledTime.format(
                        'h:mma',
                    )}</span></th>`,
                );
                if (this.selectedTimeslot === slots.timeslot) {
                    // timeslot currently selected for sign up or update
                    headerElem.addClass('appointment-slots-header-selected');
                } else if (
                    this.myRequest &&
                    this.myRequest.timeslot !== this.selectedTimeslot &&
                    this.myRequest.timeslot === slots.timeslot
                ) {
                    // current appointment pending cancel
                    headerElem.addClass('appointment-slots-header-cancel');
                } else if (
                    slots.scheduledTime.format('h:mma').indexOf('00') !== -1
                ) {
                    // on the hour
                    headerElem.addClass('appointment-slots-header-major');
                } else if (i === 0 || appointments[i - 1].numAvailable === 0) {
                    // first timeslot or first after a gap
                    headerElem.addClass('appointment-slots-header-major');
                } else {
                    // otherwise, it's minor
                    headerElem.addClass('appointment-slots-header-minor');
                }
            } else {
                headerElem = $(
                    `<th class="appointment-slots-header"><span>&nbsp</span></th>`,
                );
            }
            this.appointmentHeaderElemsMap[slots.timeslot] = headerElem;
            headerElems.push(headerElem);
            headerRow.append(headerElem);
        });

        for (let r = 1; r <= maxAppts; ++r) {
            let row = $('<tr></tr>').appendTo(this.appointmentsSlotsTable);
            appointments.forEach((slots, i) => {
                let rowElem =
                    slots.numAvailable < r
                        ? $('<td>&nbsp</td>')
                        : this.myRequest?.timeslot === slots.timeslot && r === 1
                        ? $(
                              '<td><button type="button" class="btn btn-primary">&nbsp</button></td>',
                          )
                        : slots.numFilled >= r
                        ? $(
                              '<td><button type="button" class="btn btn-danger" disabled>&nbsp</button></td>',
                          )
                        : slots.scheduledTime.diff(now) < 0
                        ? $(
                              '<td><button type="button" class="btn btn-basic" disabled>&nbsp</button></td>',
                          )
                        : $(
                              '<td><button type="button" class="btn btn-success">&nbsp</button></td>',
                          );

                rowElem
                    .find('button')
                    .hover(
                        () => {
                            headerElems[i].addClass(
                                'appointment-slots-header-hover',
                            );
                        },
                        () => {
                            headerElems[i].removeClass(
                                'appointment-slots-header-hover',
                            );
                        },
                    )
                    .click(() => {
                        this.setSelectedTimeslot(slots.timeslot);
                        this.formChanged();
                    });

                row.append(rowElem);
            });
        }
    }

    private setSelectedTimeslot(this: SignUpForm<true>, timeslot: number) {
        if (this.selectedTimeslot !== undefined) {
            this.appointmentHeaderElemsMap[this.selectedTimeslot]?.removeClass(
                'appointment-slots-header-selected',
            );
        }
        this.selectedTimeslot = timeslot;
        this.myRequest &&
            this.myRequest.timeslot !== timeslot &&
            this.appointmentHeaderElemsMap[this.myRequest.timeslot]?.addClass(
                'appointment-slots-header-cancel',
            );
        this.appointmentHeaderElemsMap[timeslot]
            ?.removeClass('appointment-slots-header-cancel')
            .addClass('appointment-slots-header-selected');
    }

    public formChanged() {
        if (this.myRequest) {
            this.formHasChanges = true;
            this.updateRequestButtons.removeClass('btn-success');
            this.updateRequestButtons.addClass('btn-warning');
            this.updateRequestButtons.prop('disabled', false);
            this.updateRequestButtons.html(this.signUpButtonContent.update);
        }
    }

    public setSignUpEnabled(isEnabled: boolean) {
        this.signUpButtons.prop('disabled', !isEnabled);

        if (this.myRequest) {
            this.updateRequestButtons.show();
            this.removeRequestButtons.show();
        } else {
            this.updateRequestButtons.hide();
            this.removeRequestButtons.hide();
        }
    }
}

class AdminControls {
    private static _name = 'AdminControls';

    private queue: OrderedQueue;
    private elem: JQuery;

    constructor(queue: OrderedQueue, elem: JQuery) {
        this.queue = queue;
        this.elem = elem;

        this.elem.append('<p><b>Admin Controls</b></p>');
        var clearQueueButton = $(
            '<button type="button" class="btn btn-danger adminOnly" data-toggle="modal" data-target="#clearTheQueueDialog">Clear the queue</button>',
        );
        this.queue.page.makeActiveOnClick(clearQueueButton); // TODO I don't think this is necessary anymore. If they can click it, it should be active.
        this.elem.append(clearQueueButton);

        this.elem.append(' ');
        var openScheduleDialogButton = $(
            '<button type="button" class="btn btn-info adminOnly" data-toggle="modal" data-target="#scheduleDialog">Schedule</button>',
        );
        this.queue.page.makeActiveOnClick(openScheduleDialogButton); // TODO I don't think this is necessary anymore. If they can click it, it should be active.
        this.elem.append(openScheduleDialogButton);

        this.elem.append(' ');
        var openManageQueueDialogButton = $(
            '<button type="button" class="btn btn-info adminOnly" data-toggle="modal" data-target="#manageQueueDialog">Manage Queue</button>',
        );
        this.queue.page.makeActiveOnClick(openManageQueueDialogButton); // TODO I don't think this is necessary anymore. If they can click it, it should be active.
        this.elem.append(openManageQueueDialogButton);

        this.elem.append(' ');
        let openAddAnnouncementDialogButton = $(
            '<button type="button" class="btn btn-info adminOnly" data-toggle="modal" data-target="#addAnnouncementDialog">Add Announcement</button>',
        );
        this.queue.page.makeActiveOnClick(openAddAnnouncementDialogButton); // TODO I don't think this is necessary anymore. If they can click it, it should be active.
        this.elem.append(openAddAnnouncementDialogButton);
    }
}

class QueueEntry {
    public readonly kind = 'queue_entry';

    private queue: OrderedQueue;

    public readonly id: string;
    public readonly timestamp: Moment;
    public readonly email: string;
    public readonly index: number;
    public readonly name: string;
    public readonly isMe: boolean;
    public readonly location?: string;
    public readonly description?: string;
    public readonly tag?: string;
    public readonly mapX?: number;
    public readonly mapY?: number;

    private elem: JQuery;
    private nameElem: JQuery;
    private locationElem?: JQuery;
    private descriptionElem?: JQuery;
    private tagElem?: JQuery;
    private tsElem: JQuery;
    private mapElem?: JQuery;
    private mapPin?: JQuery;

    constructor(
        queue: OrderedQueue,
        data: { [index: string]: string },
        index: number,
        elem: JQuery,
    ) {
        this.queue = queue;

        this.id = data['id'];
        this.timestamp = moment(data['id_timestamp']);
        this.email = data['email'];

        this.index = index;
        this.isMe = !!data['name']; // if it has a name it's them

        this.elem = elem;

        let infoElem = $('<div class="queue-entryInfo"></div>');

        let name = data['name']
            ? data['name'] + ' (' + data['email'] + ')'
            : 'Anonymous Student';
        this.nameElem = $(
            '<p><span class="glyphicon glyphicon-education"></span></p>',
        )
            .append(' ' + name)
            .appendTo(infoElem);
        if (data['tag'] && data['tag'].length > 0) {
            this.tag = data['tag'];
            this.nameElem.append(
                ' <span class="label label-info">' + this.tag + '</span>',
            );
        }
        this.name = data['name'];

        if (data['description'] && data['description'].length > 0) {
            this.descriptionElem = $(
                '<p><span class="glyphicon glyphicon-question-sign"></span></p>',
            )
                .append(' ' + data['description'])
                .appendTo(infoElem);
            this.description = data['description'];
        }

        if (data['location'] && data['location'].length > 0) {
            this.locationElem = $(
                '<p><span class="glyphicon glyphicon-map-marker"></span></p>',
            )
                .append(' ' + data['location'])
                .appendTo(infoElem);
            this.location = data['location'];
        }

        let timeWaiting = +new Date() - +this.timestamp;
        let minutesWaiting = Math.round(timeWaiting / 1000 / 60);
        this.tsElem = $('<p><span class="glyphicon glyphicon-time"></span></p>')
            .append(' ' + minutesWaiting + ' min')
            .appendTo(infoElem);

        let removeButton = $(
            '<button type="button" class="btn btn-danger">Remove</button>',
        );
        if (!this.isMe) {
            removeButton.addClass('adminOnly');
        }

        removeButton.on(
            'click',
            this.queue.removeRequest.bind(this.queue, this),
        );
        infoElem.append(removeButton);

        infoElem.append(' ');

        let sendMessageButton = $(
            '<button type="button" class="btn btn-warning adminOnly">Message</button>',
        );
        let self = this;
        sendMessageButton.on('click', function () {
            let sendMessageDialog = $('#sendMessageDialog');
            sendMessageDialog.modal('show');
            QueueApplication.instance.setSendMessageInfo(
                self.queue.page.queueId,
                self.email,
            );
        });
        infoElem.append(sendMessageButton);

        if (
            this.queue.page.hasMap() &&
            data['mapX'] !== undefined &&
            data['mapY'] !== undefined
        ) {
            let mapX = (this.mapX = parseFloat(data['mapX']));
            let mapY = (this.mapY = parseFloat(data['mapY']));

            let mapElem = $(
                '<div class="adminOnly" style="display:inline-block; vertical-align: top; width: 25%; margin-right: 10px"></div>',
            );
            this.elem.append(mapElem);

            let mapHolder = $('<div style="position: relative"></div>');
            this.mapElem = $(
                '<img class="adminOnly queue-entryMap" src="img/' +
                    this.queue.page.mapImageSrc +
                    '"></img>',
            );
            mapHolder.append(this.mapElem);
            this.mapPin = $(
                '<span class="adminOnly queue-locatePin"><span class="glyphicon glyphicon-map-marker" style="position:absolute; left:-1.3ch;top:-0.95em;"></span></span>',
            );
            this.mapPin.css('left', mapX + '%');
            this.mapPin.css('top', mapY + '%');
            mapHolder.append(this.mapPin);
            mapElem.append(mapHolder);
        } else {
            // let dibsButton = $('<button type="button" class="btn btn-info adminOnly">Dibs!</button>');
            // this.elem.append(dibsButton);
            // this.elem.append(" ");
        }

        this.elem.append(infoElem);
    }

    public onRemove() {
        // this.send("removed");
        this.elem.slideUp(ANIMATION_DELAY, function () {
            $(this).remove();
        });
    }
}

export class Schedule {
    private static readonly _name: 'Schedule';

    private static readonly sequence = {
        o: 'c',
        c: 'p',
        p: 'o',
    };

    private readonly unitElems: JQuery[][];

    constructor(elem: JQuery) {
        let dialog = $('#scheduleDialog');

        $('#scheduleForm').submit((e) => {
            e.preventDefault();

            this.update();

            dialog.modal('hide');
            return false;
        });

        dialog.on('shown.bs.modal', () => {
            this.refresh();
        });

        // Set up table in schedule picker
        let schedulePicker = $('#schedulePicker');

        // First row of table with time headers
        let firstRow = $('<tr></tr>').appendTo(schedulePicker);

        // Extra blank in first row to correspond to row labels in other rows
        firstRow.append('<td style="width:1em; padding-right: 3px;"></td>');

        for (var i = 0; i < 24; ++i) {
            firstRow.append(
                '<td colspan="2">' +
                    (i === 0 || i === 12 ? 12 : i % 12) +
                    '</td>',
            );
        }

        this.unitElems = [];
        let dayLetters = ['S', 'M', 'T', 'W', 'T', 'F', 'S'];
        for (var r = 0; r < 7; ++r) {
            var day: JQuery[] = [];
            var rowElem = $('<tr></tr>');
            rowElem.append(
                '<td style="width:1em; text-align: right; padding-right: 3px;">' +
                    dayLetters[r] +
                    '</td>',
            );
            for (var c = 0; c < 48; ++c) {
                var unitElem = $('<td><div class="scheduleUnit"></div></td>')
                    .appendTo(rowElem)
                    .find('.scheduleUnit');
                day.push(unitElem);
            }
            this.unitElems.push(day);
            schedulePicker.append(rowElem);
        }

        let pressed = false;
        schedulePicker.on('mousedown', function (e) {
            e.preventDefault();
            pressed = true;
            return false;
        });
        schedulePicker.on('mouseup', function () {
            pressed = false;
        });
        schedulePicker.on('mouseleave', function () {
            pressed = false;
        });
        dialog.on('hidden.bs.modal', function () {
            pressed = false;
        });

        let changeColor = (elem: JQuery) => {
            if (pressed) {
                var currentType: 'o' | 'c' | 'p' = elem.data('scheduleType');
                elem.removeClass('scheduleUnit-' + currentType);

                var nextType = Schedule.sequence[currentType];
                elem.data('scheduleType', nextType);
                elem.addClass('scheduleUnit-' + nextType);
            }
        };
        schedulePicker.on('mouseover', '.scheduleUnit', function (e) {
            e.preventDefault();
            changeColor($(this));
            return false;
        });
        schedulePicker.on('mousedown', '.scheduleUnit', function (e) {
            e.preventDefault();
            pressed = true;
            changeColor($(this));
            return false;
        });
    }

    public refresh() {
        let aq = QueueApplication.instance.activeQueue();
        if (aq) {
            return $.ajax({
                type: 'GET',
                url: 'api/queues/' + aq.queueId + '/schedule',
                dataType: 'json',
                success: (data) => {
                    var schedule = data; // array of 7 strings
                    for (var r = 0; r < 7; ++r) {
                        for (var c = 0; c < 48; ++c) {
                            var elem = this.unitElems[r][c];
                            elem.removeClass(); // removes all classes
                            elem.addClass('scheduleUnit');
                            elem.addClass(
                                'scheduleUnit-' + schedule[r].charAt(c),
                            );
                            elem.data('scheduleType', schedule[r].charAt(c));
                        }
                    }
                },
                error: oops,
            });
        }
    }

    public update() {
        if (!QueueApplication.instance.activeQueue()) {
            return;
        }

        // lol can't make up my mind whether I like functional vs. iterative
        var schedule = [];
        for (var r = 0; r < 7; ++r) {
            schedule.push(
                this.unitElems[r]
                    .map(function (unitElem) {
                        return unitElem.data('scheduleType');
                    })
                    .join(''),
            );
        }

        let aq = QueueApplication.instance.activeQueue();
        if (aq) {
            return $.ajax({
                type: 'PUT',
                url: 'api/queues/' + aq.queueId + '/schedule',
                data: JSON.stringify(schedule),
                success: function () {
                    console.log('schedule updated');
                },
                error: oops,
            });
        }
    }
}

export class ManageQueueDialog {
    private static readonly _name: 'ManageQueueDialog';

    private static readonly POLICIES_UP_TO_DATE =
        '<span><span class="glyphicon glyphicon-floppy-saved"></span> Saved</span>';
    private static readonly POLICIES_UNSAVED =
        '<span><span class="glyphicon glyphicon-floppy-open"></span> Update Configuration</span>';

    public readonly _act!: MessageResponses;

    private readonly updateConfigurationButton: JQuery;

    constructor() {
        let dialog = $('#manageQueueDialog');

        let groupsForm = $('#groupsForm');
        groupsForm.submit(function (e) {
            e.preventDefault();
            var file = $('#groups-upload').prop('files')[0];
            var reader = new FileReader();
            reader.onload = () => {
                let aq = QueueApplication.instance.activeQueue();
                aq && aq.updateGroups(reader.result);
                return false;
            };
            reader.readAsText(file);
        });

        let policiesForm = $('#policiesForm');
        policiesForm.submit((e) => {
            e.preventDefault();

            this.update();

            return false;
        });

        this.updateConfigurationButton = $('#updateConfigurationButton');

        $('#preventUnregisteredCheckbox').change(
            this.unsavedChanges.bind(this),
        );
        $('#preventGroupsCheckbox').change(this.unsavedChanges.bind(this));
        $('#prioritizeNewCheckbox').change(this.unsavedChanges.bind(this));
        $('#preventGroupsBoostCheckbox').change(this.unsavedChanges.bind(this));

        QueueApplication.instance.observable.addListener(this);
        this.refresh();
    }

    @messageResponse('activeQueueSet')
    public refresh() {
        let aq = QueueApplication.instance.activeQueue();
        if (!aq) {
            return;
        }
        if (!aq.isAdmin) {
            return;
        }

        $('#checkQueueRosterLink').attr(
            'href',
            `api/queues/${aq.queueId}/roster`,
        );
        $('#checkQueueGroupsLink').attr(
            'href',
            `api/queues/${aq.queueId}/groups`,
        );

        return $.ajax({
            type: 'GET',
            url: 'api/queues/' + aq.queueId + '/configuration',
            dataType: 'json',
            success: this.refreshResponse.bind(this),
            error: oops,
        });
    }

    private refreshResponse(data: { [index: string]: string }) {
        console.log(JSON.stringify(data));
        $('#preventUnregisteredCheckbox').prop(
            'checked',
            data['prevent_unregistered'],
        );
        $('#preventGroupsCheckbox').prop('checked', data['prevent_groups']);
        $('#prioritizeNewCheckbox').prop('checked', data['prioritize_new']);
        $('#preventGroupsBoostCheckbox').prop(
            'checked',
            data['prevent_groups_boost'],
        );

        this.changesUpToDate();
    }

    public update() {
        let aq = QueueApplication.instance.activeQueue();
        if (!aq) {
            return;
        }
        aq.updateConfiguration({
            prevent_unregistered: $('#preventUnregisteredCheckbox').is(
                ':checked',
            ),
            prevent_groups: $('#preventGroupsCheckbox').is(':checked'),
            prioritize_new: $('#prioritizeNewCheckbox').is(':checked'),
            prevent_groups_boost: $('#preventGroupsBoostCheckbox').is(
                ':checked',
            ),
        }).done(this.changesUpToDate.bind(this));
    }

    private unsavedChanges() {
        this.updateConfigurationButton
            .html(ManageQueueDialog.POLICIES_UNSAVED)
            .prop('disabled', false)
            .removeClass('btn-success')
            .addClass('btn-warning');
    }

    private changesUpToDate() {
        this.updateConfigurationButton
            .html(ManageQueueDialog.POLICIES_UP_TO_DATE)
            .prop('disabled', true)
            .removeClass('btn-warning')
            .addClass('btn-success');
    }
}

export function filterAppointmentsSchedule(appointments: AppointmentSchedule) {
    appointments = appointments.filter(
        (slots, i) =>
            slots.numAvailable > 0 ||
            (i !== 0 && appointments[i - 1].numAvailable > 0),
    );
    // if last filtered appointment is empty, pop it
    if (appointments.length > 0 && appointments[appointments.length - 1].numAvailable === 0) {
        appointments.pop();
    }
    return appointments;
}
