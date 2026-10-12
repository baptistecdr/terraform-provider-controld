// Copyright (c) baptistecdr
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	controld "github.com/baptistecdr/controld-go"
)

var (
	_ resource.Resource                = &EndpointScheduleResource{}
	_ resource.ResourceWithConfigure   = &EndpointScheduleResource{}
	_ resource.ResourceWithImportState = &EndpointScheduleResource{}
)

var validScheduleWeekdays = []string{"mon", "tue", "wed", "thu", "fri", "sat", "sun"}

func NewEndpointScheduleResource() resource.Resource {
	return &EndpointScheduleResource{}
}

// EndpointScheduleResource manages a weekly schedule attached to a device,
// during which another profile is enforced.
type EndpointScheduleResource struct {
	client *controld.API
}

type EndpointScheduleResourceModel struct {
	ID       types.String                  `tfsdk:"id"`
	Name     types.String                  `tfsdk:"name"`
	DeviceID types.String                  `tfsdk:"device_id"`
	TimeZone types.String                  `tfsdk:"time_zone"`
	Status   types.Bool                    `tfsdk:"status"`
	Windows  []EndpointScheduleWindowModel `tfsdk:"windows"`
}

type EndpointScheduleWindowModel struct {
	Weekday   types.String `tfsdk:"weekday"`
	TimeStart types.String `tfsdk:"time_start"`
	TimeEnd   types.String `tfsdk:"time_end"`
	ProfileID types.String `tfsdk:"profile_id"`
}

func (r *EndpointScheduleResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_endpoint_schedule"
}

func (r *EndpointScheduleResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages an endpoint schedule: a weekly set of time windows attached to a device, during which another profile is enforced on that device. " +
			"This relies on an undocumented ControlD API endpoint (`/endpointschedules`), so its behaviour may change without notice.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The schedule identifier (PK).",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The name of the schedule.",
			},
			"device_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The identifier of the device the schedule is attached to. Changing this forces a new resource to be created.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"time_zone": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The IANA time zone the windows are expressed in, e.g. `Europe/Paris`.",
			},
			"status": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Whether the schedule is enabled. Defaults to enabled.",
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"windows": schema.ListNestedAttribute{
				Required:            true,
				MarkdownDescription: "The time windows of the schedule. Windows on the same day cannot overlap.",
				Validators: []validator.List{
					listvalidator.SizeAtLeast(1),
				},
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"weekday": schema.StringAttribute{
							Required:            true,
							MarkdownDescription: "The day of the week the window applies to. One of: `mon`, `tue`, `wed`, `thu`, `fri`, `sat`, `sun`.",
							Validators: []validator.String{
								stringvalidator.OneOf(validScheduleWeekdays...),
							},
						},
						"time_start": schema.StringAttribute{
							Required:            true,
							MarkdownDescription: "When the window starts, in 24h `HH:MM` format.",
						},
						"time_end": schema.StringAttribute{
							Required:            true,
							MarkdownDescription: "When the window ends, in 24h `HH:MM` format.",
						},
						"profile_id": schema.StringAttribute{
							Required:            true,
							MarkdownDescription: "The identifier of the profile enforced on the device during the window.",
						},
					},
				},
			},
		},
	}
}

func (r *EndpointScheduleResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	client, ok := configureClient(req.ProviderData, &resp.Diagnostics)
	if !ok {
		return
	}
	r.client = client
}

func (r *EndpointScheduleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data EndpointScheduleResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	params := controld.CreateEndpointScheduleParams{
		Name:     data.Name.ValueString(),
		DeviceID: data.DeviceID.ValueString(),
		TimeZone: data.TimeZone.ValueString(),
		Windows:  windowParamsFromModel(data.Windows),
	}
	if !data.Status.IsNull() && !data.Status.IsUnknown() {
		v := controld.IntBool(data.Status.ValueBool())
		params.Status = &v
	}

	schedule, err := r.client.CreateEndpointSchedule(ctx, params)
	if err != nil {
		resp.Diagnostics.AddError("Unable to Create Endpoint Schedule", err.Error())
		return
	}

	updateModelFromSchedule(&data, schedule)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *EndpointScheduleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data EndpointScheduleResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// There is no single-schedule endpoint: GET /endpointschedules/{id} just
	// returns the full list.
	schedules, err := r.client.ListEndpointSchedules(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Unable to Read Endpoint Schedule", err.Error())
		return
	}

	var found *controld.EndpointSchedule
	for i := range schedules {
		if schedules[i].PK == data.ID.ValueString() {
			found = &schedules[i]
			break
		}
	}
	if found == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	updateModelFromSchedule(&data, *found)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *EndpointScheduleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data, state EndpointScheduleResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// The update is a true partial update: only send what changed.
	params := controld.UpdateEndpointScheduleParams{
		ScheduleID: data.ID.ValueString(),
	}
	if !data.Name.Equal(state.Name) {
		v := data.Name.ValueString()
		params.Name = &v
	}
	if !data.TimeZone.Equal(state.TimeZone) {
		v := data.TimeZone.ValueString()
		params.TimeZone = &v
	}
	if !data.Status.IsNull() && !data.Status.IsUnknown() && !data.Status.Equal(state.Status) {
		v := controld.IntBool(data.Status.ValueBool())
		params.Status = &v
	}
	// Windows, when sent, replace all the windows of the schedule.
	if !windowsEqual(data.Windows, state.Windows) {
		params.Windows = windowParamsFromModel(data.Windows)
	}

	schedule, err := r.client.UpdateEndpointSchedule(ctx, params)
	if err != nil {
		resp.Diagnostics.AddError("Unable to Update Endpoint Schedule", err.Error())
		return
	}

	updateModelFromSchedule(&data, schedule)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *EndpointScheduleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data EndpointScheduleResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.client.DeleteEndpointSchedule(ctx, controld.DeleteEndpointScheduleParams{
		ScheduleID: data.ID.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Unable to Delete Endpoint Schedule", err.Error())
		return
	}
}

func (r *EndpointScheduleResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
}

func windowParamsFromModel(windows []EndpointScheduleWindowModel) []controld.EndpointScheduleWindowParams {
	params := make([]controld.EndpointScheduleWindowParams, 0, len(windows))
	for _, w := range windows {
		params = append(params, controld.EndpointScheduleWindowParams{
			Weekday:   controld.ScheduleWeekday(w.Weekday.ValueString()),
			TimeStart: w.TimeStart.ValueString(),
			TimeEnd:   w.TimeEnd.ValueString(),
			ProfileID: w.ProfileID.ValueString(),
		})
	}
	return params
}

func windowsEqual(a, b []EndpointScheduleWindowModel) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if !a[i].Weekday.Equal(b[i].Weekday) ||
			!a[i].TimeStart.Equal(b[i].TimeStart) ||
			!a[i].TimeEnd.Equal(b[i].TimeEnd) ||
			!a[i].ProfileID.Equal(b[i].ProfileID) {
			return false
		}
	}
	return true
}

func updateModelFromSchedule(data *EndpointScheduleResourceModel, schedule controld.EndpointSchedule) {
	data.ID = types.StringValue(schedule.PK)
	data.Name = types.StringValue(schedule.Name)
	data.TimeZone = types.StringValue(schedule.TimeZone)
	data.Status = types.BoolValue(bool(schedule.Status))
	// Device is only partially populated, and absent from create/update
	// responses; keep the configured value unless it is unknown (import).
	if schedule.Device.PK != "" && (data.DeviceID.IsNull() || data.DeviceID.IsUnknown()) {
		data.DeviceID = types.StringValue(schedule.Device.PK)
	}

	data.Windows = make([]EndpointScheduleWindowModel, 0, len(schedule.Windows))
	for _, w := range schedule.Windows {
		data.Windows = append(data.Windows, EndpointScheduleWindowModel{
			Weekday:   types.StringValue(string(w.Weekday)),
			TimeStart: types.StringValue(w.TimeStart),
			TimeEnd:   types.StringValue(w.TimeEnd),
			ProfileID: types.StringValue(w.ProfileID),
		})
	}
}
