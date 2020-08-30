/**
 * Created by James Juett on 9/5/2016.
 */

import { Observable, MessageResponses, messageResponse } from './util/mixins';
import escape from 'lodash/escape';
import endsWith from 'lodash/endsWith';
import { Mutable, oops, showErrorMessage } from './util/util';
import { QueueKind, Page } from './queue';
import { OrderedQueue } from './OrderedQueue';
import { AppointmentsQueue } from './AppointmentsQueue';
import $ from 'jquery';

export class QueueApplication {
    private static _name = 'QueueApplication';

    public static readonly instance: QueueApplication;

    public static createInstance(elem: JQuery) {
        (<QueueApplication>QueueApplication.instance) = new QueueApplication(
            elem,
        );
    }

    private elem: JQuery;
    private coursePills: JQuery;
    private coursePanes: JQuery;

    private courses: Course[] = [];
    public readonly activeCourse?: Course;
    private _activePage?: Page;

    private messagesShown: { [index: string]: boolean } = {};
    private sendMessageQueueId?: string;
    private sendMessageReceiver?: string;

    public readonly observable = new Observable(this);

    private constructor(elem: JQuery) {
        this.elem = elem;
        this.coursePills = elem.find('.coursePills');
        this.coursePanes = elem.find('.coursePanes');

        this.loadCourses();
    }

    public async loadCourses() {
        try {
            let list = await $.getJSON('api/courses');
            this.onCoursesLoad(list);
        } catch (e) {
            oops(e, e);
        }
    }

    public onCoursesLoad(list: { [index: string]: string }[]) {
        this.coursePills.empty();
        this.coursePanes.empty();
        this.courses.clear();

        // No active course initially
        this.coursePanes.append(
            $(
                '<div class="tab-pane fade in active"><h1><span class="glyphicon glyphicon-arrow-left"></span> Please select a course.</h1></div>',
            ),
        );

        list.forEach((courseData) => {
            // Escape everything
            for (var key in courseData) {
                courseData[key] = escape(courseData[key]);
            }

            let queueId = courseData['id'];
            let shortName = courseData['short_name'];

            // Add the pill used to select the course
            let pillElem = $(
                '<li><a href="#' +
                    queueId +
                    '" data-toggle="pill"><h3>' +
                    shortName +
                    '</h3></a></li>',
            );
            this.coursePills.append(pillElem);

            // Add the element that will contain the course content
            let courseElem = $(
                '<div id="' + queueId + '" class="tab-pane fade"></div>',
            );
            this.coursePanes.append(courseElem);

            // Create the course itself
            let course = new Course(courseData, courseElem);
            this.courses.push(course);

            pillElem.find('a').click(() => {
                course.makeActive();
                (<Mutable<this>>this).activeCourse = course;
            });
        });
    }

    public setActiveQueue(queue: any) {
        this._activePage = queue;
        console.log('Setting active queue to ' + queue.queueId);
        this.updateSignUpForm();
        this.observable.send('activeQueueSet');
    }

    public activeQueue() {
        return this._activePage;
    }

    public updateSignUpForm() {
        if (this._activePage && this._activePage.hasMap()) {
            $('#signUpMapHolder').show();
            $('#signUpMapMessage').show();
            $('#signUpMapImage').attr('src', this._activePage.mapImageSrc);
        } else {
            $('#signUpMapHolder').hide();
            $('#signUpMapMessage').hide();
        }
    }

    public userSignedIn() {

        if (User.isSuper()) {
            $("body").removeClass("notSuper");
        }
        else {
            $("body").addClass("notSuper");
        }

        this.courses.forEach((course) => {
            course.userSignedIn();
        });
    }

    public refreshActivePage() {
        this._activePage && this._activePage.refresh();
        this.refreshContent();
    }

    public message(message: { [index: string]: string }) {
        if (!this.messagesShown[message.id]) {
            this.messagesShown[message.id] = true;
            $('#messageDialogHeader').html('Message');
            $('#messageDialogContent').append(
                '<p><span class="label label-info">' +
                    message['sender'] +
                    '</span> ' +
                    message['content'] +
                    '</p>',
            );
            $('#messageDialog').modal('show');
        }
    }

    public setSendMessageInfo(queueId: string, receiver: string) {
        this.sendMessageQueueId = queueId;
        this.sendMessageReceiver = receiver;
    }

    public sendMessage(content: string) {
        $.ajax({
            type: 'POST',
            url: `api/queues/${this.sendMessageQueueId}/messages`,
            data: JSON.stringify({
                content: content,
                receiver: this.sendMessageReceiver,
            }),
            contentType: 'application/json',
            success: function () {},
            error: (xhr) => {
                if (!xhr) {
                    oops(xhr);
                    return;
                }

                showErrorMessage(JSON.parse(xhr.responseText)['message']);
            },
        });
    }

    public refreshContent() {
        if (this._activePage) {
            let title = this._activePage.course.shortName + ' OH';
            if (this._activePage.queue instanceof OrderedQueue) {
                title += ' (' + this._activePage.queue.numEntries + ')';
            }
            document.title = title;
        }
    }

    public notify(title: string, message: string) {
        if (!Notification) {
            alert(message);
        } else {
            if (Notification.permission !== 'granted') {
                Notification.requestPermission();
            } else {
                new Notification(title, {
                    body: message,
                });
            }
        }
    }

    public createCourse(courseSpec: NewCourseSpecification) {
        return fetch("api/courses/", {
            method: "POST",
            body: JSON.stringify(courseSpec)
        });
    }
}

export class Course {
    public readonly courseId: string;
    public readonly shortName: string;
    public readonly fullName: string;

    private isAdmin: boolean = false;
    private queues: any[] = [];
    private activeQueue: any;

    private readonly courseAdminControls: CourseAdminControls;

    private readonly elem: JQuery;
    private readonly queuePillsElem: JQuery;
    private readonly pickAQueueElem: JQuery;
    private readonly mainElem: JQuery;
    private readonly queuePanesElem: JQuery;
    private readonly contentElem: JQuery;

    constructor(data: { [index: string]: string }, elem: JQuery) {
        this.courseId = data['id'];
        this.shortName = data['short_name'];
        this.fullName = data['full_name'];

        this.elem = elem;

        this.courseAdminControls = new CourseAdminControls(
            this, $('<div></div>').appendTo(this.elem)
        );

        this.queuePillsElem = $('<ul class="queuePills nav nav-pills"></ul>');
        this.elem.append(this.queuePillsElem);

        this.pickAQueueElem = $('<div></div>');
        this.pickAQueueElem.append(
            $(
                '<h3><span class="glyphicon glyphicon-arrow-up"></span> Several queues are available for ' +
                    this.shortName +
                    '. Please select one.</h3>',
            ),
        );
        this.elem.append(this.pickAQueueElem);

        this.mainElem = $('<div></div>');
        this.mainElem.hide();

        this.queuePanesElem = $(
            '<div class="col-xs-12 col-md-12 queuePanes tab-content"></div>',
        );
        this.mainElem.append(this.queuePanesElem);

        this.contentElem = $('<div class="col-xs-12 col-md-12"></div>');
        this.mainElem.append(this.contentElem);

        this.elem.append(this.mainElem);

        this.loadQueues();
    }

    public makeActive() {
        // Don't need to do anything in particular for the course itself,
        // but we do need to make sure the active queue within this course
        // is the active queue overall since it will be shown.
        this.activeQueue && this.activeQueue.makeActive();
    }

    private loadQueues() {
        return $.ajax({
            type: 'GET',
            url: 'api/courses/' + this.courseId + '/queues',
            dataType: 'json',
            success: (data) => {
                this.onQueuesLoad(data);
            },
            error: (xhr) => {
                if (!xhr) {
                    oops(xhr);
                    return;
                }

                showErrorMessage(JSON.parse(xhr.responseText)['message']);
            },
        });
    }

    private onQueuesLoad(list: any[]) {
        this.queues.clear();
        this.queuePillsElem.empty();
        this.queuePanesElem.empty();

        list.forEach((item) => {
            var name = item['name'];
            var queueId = item['id'];

            // Add pills for each queue belonging to this course
            var pillElem = $(
                '<li><a data-toggle="pill"><h6>' + name + '</h6></a></li>',
            );
            pillElem.find('a').prop('href', '#queue' + queueId);
            this.queuePillsElem.append(pillElem);

            // Add panes to hold the queue
            var queueElem = $('<div id="queue' + queueId + '"></div>');
            queueElem.addClass('tab-pane fade');
            this.queuePanesElem.append(queueElem);

            // Create the queue objects themselves
            var queue = new Page(item, this, item['type'], queueElem);
            this.queues.push(queue);

            queue.refresh();

            pillElem.find('a').click(() => {
                this.pickAQueueElem.empty();
                this.activeQueue = queue;
                this.mainElem.show();
                queue.makeActive();
            });
        });

        // If only one queue, select it automatically
        // (pillElem and queueElem are still in scope even after the loop body)
        if (this.queues.length === 1) {
            this.queuePillsElem.children().first().addClass('active');
            this.queuePanesElem.children().first().addClass('in active');
            this.activeQueue = this.queues[0];
            this.pickAQueueElem.hide();
            this.mainElem.show();
        } else {
            this.pickAQueueElem.show();
            this.mainElem.hide();
        }

        this.setAdmin(User.isCourseAdmin(this.courseId));
    }

    public setAdmin(isAdmin: boolean) {
        this.isAdmin = isAdmin;
        for (var i = 0; i < this.queues.length; ++i) {
            this.queues[i].setAdmin(isAdmin);
        }
        if (this.isAdmin) {
            this.elem.addClass('admin');
            this.elem.removeClass('notAdmin');
        } else {
            this.elem.addClass('notAdmin');
            this.elem.removeClass('admin');
        }
    }

    public userSignedIn() {
        this.setAdmin(User.isCourseAdmin(this.courseId));
        this.queues.forEach(function (queue) {
            queue.userSignedIn();
        });
    }

    public createQueue(queueSpec: NewQueueSpecification) {
        return fetch(`api/courses/${this.courseId}/queues`, {
            method: "POST",
            body: JSON.stringify(queueSpec)
        });
    }
    
    public getStaff() {
        return fetch(`api/courses/${this.courseId}/admins`, {
            method: "GET"
        });
    }

    public setStaff(staffSpec: CourseStaffSpecification) {
        
        // filter to unique staff
        staffSpec = Array.from(new Set(staffSpec));

        return fetch(`api/courses/${this.courseId}/admins`, {
            method: "PUT",
            body: JSON.stringify(staffSpec)
        });
    }
}

class CourseAdminControls {
    private course: Course;
    private elem: JQuery;

    constructor(course: Course, elem: JQuery) {
        this.course = course;
        this.elem = elem.addClass("adminOnly");

        this.elem.append('<p><b>Course Admin</b></p>');

        this.elem.append('<button type=" button" class="btn btn-info adminOnly" data-toggle="modal" data-target="#createQueueDialog">Create Queue</button>');
        this.elem.append(" ");
        this.elem.append('<button type=" button" class="btn btn-info adminOnly" data-toggle="modal" data-target="#editStaffDialog">Edit Course Staff</button>');

        this.elem.append('<hr />')
    }
}

interface NewCourseSpecification {
    readonly short_name: string;
    readonly full_name: string;
}

export class CreateCourseDialog {

    public constructor() {
        let createCourseForm = $('#createCourseForm');
        createCourseForm.submit(async (e) => {
            e.preventDefault();
            
            try {
                let response = await QueueApplication.instance.createCourse({
                    short_name: ""+$("#shortNameInput").val(),
                    full_name: ""+$("#fullNameInput").val()
                });

                if (!response.ok) {
                    showErrorMessage(JSON.parse(await response.text()));
                }
            }
            catch(e) {
                showErrorMessage(e);
            }
            $('#createCourseDialog').modal('hide');
        });
    }

}



interface NewQueueSpecification {
    readonly type: QueueKind;
    readonly name: string;
    readonly location: string;
    readonly map: string;
    readonly active: boolean;
}

export class CreateQueueDialog {

    public constructor() {
        let createQueueForm = $('#createQueueForm');
        createQueueForm.submit(async (e) => {
            e.preventDefault();
            
            try {
                let response = await QueueApplication.instance.activeCourse!.createQueue({
                    type: <QueueKind>(""+$('#createQueueForm input[name="type"]:checked').val()),
                    name: ""+$('#createQueueForm input[name="name"]').val(),
                    location: ""+$('#createQueueForm input[name="location"]').val(),
                    map: ""+$('#createQueueForm input[name="map"]').val(),
                    active: "true" === $('#createQueueForm input[name="active"]').val(),
                });

                if (!response.ok) {
                    showErrorMessage(JSON.parse(await response.text()));
                }
            }
            catch(e) {
                showErrorMessage(e);
            }
            $('#createQueueDialog').modal('hide');
        });
    }

}


type CourseStaffSpecification = readonly string[];

export class EditStaffDialog {

    private static readonly STAFF_UP_TO_DATE =
        '<span><span class="glyphicon glyphicon-floppy-saved"></span> Saved</span>';
    private static readonly STAFF_UNSAVED =
        '<span><span class="glyphicon glyphicon-floppy-open"></span> Update Staff</span>';

    private updateStaffButton: JQuery;

    public constructor() {
        this.updateStaffButton = $("#editStaffForm button:submit");

        $("#editStaffDialog").on('shown.bs.modal', () => {
            this.refresh();
        });

        $('#editStaffForm textarea[name="staff"]')
            .on("keyup", () => { this.unsavedChanges(); return true; });

        $("#editStaffForm").submit(async (e) => {
            e.preventDefault();
            
            try {
                let response = await QueueApplication.instance.activeCourse!.setStaff(
                    (""+$('#editStaffForm textarea[name="staff"]').val()).split("\n").map((s) => s.trim())
                );

                if (response.ok) {
                    this.changesUpToDate();
                }
                else {
                    showErrorMessage(JSON.parse(await response.text()));
                }
            }
            catch(e) {
                showErrorMessage(e);
            }
            $('#createQueueDialog').modal('hide');
        });
    }

    private async refresh() {
        try {
            let response = await QueueApplication.instance.activeCourse!.getStaff();

            if (response.ok) {
                $('#editStaffForm textarea[name="staff"]').val(
                    JSON.parse((await response.text())).join("\n")
                );
                this.changesUpToDate();
            }
            else {
                $('#createQueueDialog').modal('hide');
                showErrorMessage(JSON.parse(await response.text()));
            }
        }
        catch(e) {
            showErrorMessage(e);
            $('#createQueueDialog').modal('hide');
        }
    }

    private unsavedChanges() {
        this.updateStaffButton
            .html(EditStaffDialog.STAFF_UNSAVED)
            .prop('disabled', false)
            .removeClass('btn-success')
            .addClass('btn-warning');
    }

    private changesUpToDate() {
        this.updateStaffButton
            .html(EditStaffDialog.STAFF_UP_TO_DATE)
            .prop('disabled', true)
            .removeClass('btn-warning')
            .addClass('btn-success');
    }
}

export namespace User {
    export function signIn(email: string, idtoken: string) {
        let newUser = new AuthenticatedUser(email, idtoken);

        var accountMessageElem = $('#accountMessage');
        // If they're not umich, they can't sign up!
        if (!newUser.isUmich()) {
            accountMessageElem.show();
            accountMessageElem.html(
                'Hi ' +
                    newUser.email +
                    '! Please <a>sign out</a> and switch to an @umich.edu account to use the queue.',
            );
            accountMessageElem.find('a').click(function () {
                var auth2 = gapi.auth2.getAuthInstance();
                auth2.disconnect().then(function () {
                    User.signOut();
                    accountMessageElem.hide();
                });
            });

            $('.openSignUpDialogButton').prop('disabled', true);
        } else {
            accountMessageElem.empty();
            accountMessageElem.hide();
        }
    }

    export function signOut() {
        var accountMessageElem = $('#accountMessage');
        accountMessageElem.hide();

        theUser && theUser.onSignOut();

        new UnauthenticatedUser(); // will implicitly set theUser singleton instance
    }

    export function idToken() {
        return theUser.idToken();
    }

    export function isUmich() {
        return theUser.isUmich();
    }

    export function isCourseAdmin(courseId: string) {
        return theUser.isCourseAdmin(courseId);
    }

    export function isMe(email: string) {
        return theUser.isMe(email);
    }

    export function isSuper() {
        return theUser.isSuper;
    }

    export function email() {
        return theUser.email;
    }

    abstract class UserBase {
        private static _name = 'UserBase';

        public abstract isUmich(): boolean;
        public abstract idToken(): string;
        public abstract isCourseAdmin(courseId: string): boolean;
        public abstract isMe(email: string): boolean;
        public abstract readonly email?: string;
        public abstract readonly isSuper?: boolean;

        public onSignOut() {
            // nothing to do here for now
        }

        public onFinishSigningIn() {
            theUser = this;

            // Notify the application there's a new user in town
            QueueApplication.instance &&
                QueueApplication.instance.userSignedIn();
        }
    }

    class AuthenticatedUser extends UserBase {
        public readonly email: string;
        private readonly _idToken: string;
        private admin_courses: { [index: string]: boolean } = {};
        public readonly isSuper?: boolean;

        constructor(email: string, idtoken: string) {
            super();
            this.email = email;
            this._idToken = idtoken;

            $.ajax({
                type: 'POST',
                url: 'api/login',
                data: {
                    idtoken: this.idToken(),
                },
                success: (data) => {
                    this.checkAdmin();
                },
                error: (xhr) => {
                    if (!xhr) {
                        oops(xhr);
                        return;
                    }

                    showErrorMessage(JSON.parse(xhr.responseText)['message']);
                },
            });
        }

        public isUmich(): boolean {
            return endsWith(this.email, '@umich.edu');
        }

        public isMe(email: string): boolean {
            return this.email === email;
        }

        public idToken(): string {
            return this._idToken;
        }

        private checkAdmin(): void {
            $.ajax({
                type: 'GET',
                url: 'api/users/@me',
                dataType: 'json',
                success: (data) => {
                    (<Mutable<this>>this).isSuper = data["site_admin"];

                    let admin_courses: readonly string[] = data["admin_courses"];
                    for (var i = 0; i < admin_courses.length; ++i) {
                        this.admin_courses[admin_courses[i]] = true;
                    }

                    // TODO HACK If admin for anything, give them fast refresh
                    // should only be on the queues they administer
                    // also if admin prompt for notifications
                    if (admin_courses.length > 0) {
                        setInterval(function () {
                            QueueApplication.instance.refreshActivePage();
                        }, 5000);

                        if (Notification) {
                            Notification.requestPermission();
                        }
                    } else {
                        setInterval(function () {
                            QueueApplication.instance.refreshActivePage();
                        }, 60000);
                    }

                    this.onFinishSigningIn();
                },
                error: (xhr) => {
                    if (!xhr) {
                        oops(xhr);
                        return;
                    }

                    showErrorMessage(JSON.parse(xhr.responseText)['message']);
                },
            });
        }

        public isCourseAdmin(courseId: string): boolean {
            return this.admin_courses[courseId];
        }
    }

    class UnauthenticatedUser extends UserBase {

        public readonly isSuper = false;

        constructor() {
            super();

            this.onFinishSigningIn();

            setInterval(function () {
                QueueApplication.instance.refreshActivePage();
            }, 60000);

            // TODO: clean up where the refresh intervals get set
            //       right now it seems like multiple can get set
        }

        public isUmich(): boolean {
            return false;
        }
        public idToken(): string {
            return '';
        }
        public isCourseAdmin(courseId: string): boolean {
            return false;
        }
        public isMe(email: string): boolean {
            return false;
        }
        public readonly email: undefined;
    }

    let theUser: UserBase = new UnauthenticatedUser();
} // end User namespace

// Give warning to users in Safari/iOS private browsing
// mode that Google sign-in won't work.
// TODO: I'm not convinced this actually does anything
//https://gist.github.com/philfreo/68ea3cd980d72383c951
if (typeof sessionStorage === 'object') {
    try {
        sessionStorage.setItem('localStorage', '1');
        sessionStorage.removeItem('localStorage');
    } catch (e) {
        oops(
            null,
            'It looks like local storage is disabled in your browser. This may aggravate an issue with Google sign-in on Safari or iOS while using private browsing mode.',
        );
    }
}
