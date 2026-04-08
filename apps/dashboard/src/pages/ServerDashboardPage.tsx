import {
	Accordion,
	ActionIcon,
	Badge,
	Button,
	Card,
	CopyButton,
	Divider,
	FileInput,
	Group,
	Loader,
	NumberInput,
	Select,
	SimpleGrid,
	Stack,
	Switch,
	Table,
	Tabs,
	Text,
	Textarea,
	TextInput,
	Tooltip,
} from "@mantine/core";
import { notifications } from "@mantine/notifications";
import {
	IconArrowLeft,
	IconCheck,
	IconCopy,
	IconExternalLink,
	IconRefresh,
} from "@tabler/icons-react";
import {
	type ReactNode,
	useCallback,
	useEffect,
	useMemo,
	useState,
} from "react";
import { get, post } from "../api";
import { MetricCard } from "../components/MetricCard";
import { PageHeader } from "../components/PageHeader";
import { badgeColor } from "../format";
import type {
	AuthMe,
	GuildChannelInfo,
	GuildDashboard,
	GuildEmojiInfo,
	GuildMemberInfo,
	GuildPluginConfig,
	GuildRoleInfo,
	GuildStickerInfo,
	PluginSection,
	WarningInfo,
} from "../types";

const LEADING_HASH_RE = /^#/;
const NON_HEX_RE = /[^0-9a-f]/i;

type Props = {
	me: AuthMe | null;
	guildDashboard: GuildDashboard | null;
	guildID: string;
	csrfToken: string;
	loading: boolean;
	onLogin: () => void;
	onBack: () => void;
	onRefresh: () => void;
	onInstall: (guildID: string) => void;
};

export function ServerDashboardPage({
	me,
	guildDashboard,
	guildID,
	csrfToken,
	loading,
	onLogin,
	onBack,
	onRefresh,
	onInstall,
}: Props) {
	const [savingPlugin, setSavingPlugin] = useState<string | null>(null);
	const [channels, setChannels] = useState<GuildChannelInfo[]>([]);
	const [roles, setRoles] = useState<GuildRoleInfo[]>([]);
	const [members, setMembers] = useState<GuildMemberInfo[]>([]);
	const [emojis, setEmojis] = useState<GuildEmojiInfo[]>([]);
	const [stickers, setStickers] = useState<GuildStickerInfo[]>([]);
	const [warnings, setWarnings] = useState<WarningInfo[]>([]);
	const [assetsLoading, setAssetsLoading] = useState(false);

	const [funConfig, setFunConfig] = useState<GuildPluginConfig>(emptyConfig());
	const [infoConfig, setInfoConfig] = useState<GuildPluginConfig>(
		emptyConfig(),
	);
	const [managerConfig, setManagerConfig] = useState<GuildPluginConfig>(
		emptyConfig(),
	);
	const [moderationConfig, setModerationConfig] = useState<GuildPluginConfig>(
		emptyModerationConfig(),
	);
	const [wellnessConfig, setWellnessConfig] = useState<GuildPluginConfig>(
		emptyWellnessConfig(),
	);

	const [memberQuery, setMemberQuery] = useState("");
	const [selectedMemberID, setSelectedMemberID] = useState("");
	const [warnReason, setWarnReason] = useState("");

	const [slowmodeChannelID, setSlowmodeChannelID] = useState("");
	const [slowmodeSeconds, setSlowmodeSeconds] = useState<number | "">(0);
	const [nicknameMemberID, setNicknameMemberID] = useState("");
	const [nicknameValue, setNicknameValue] = useState("");

	const [roleMode, setRoleMode] = useState<"add" | "remove">("add");
	const [roleName, setRoleName] = useState("");
	const [roleColor, setRoleColor] = useState("");
	const [roleEditID, setRoleEditID] = useState("");
	const [roleMemberID, setRoleMemberID] = useState("");
	const [roleMemberRoleID, setRoleMemberRoleID] = useState("");

	const [purgeChannelID, setPurgeChannelID] = useState("");
	const [purgeMode, setPurgeMode] = useState("all");
	const [purgeAnchor, setPurgeAnchor] = useState("");
	const [purgeCount, setPurgeCount] = useState<number | "">(10);

	const [emojiName, setEmojiName] = useState("");
	const [emojiFile, setEmojiFile] = useState<File | null>(null);
	const [emojiEditID, setEmojiEditID] = useState("");
	const [emojiEditName, setEmojiEditName] = useState("");
	const [emojiDeleteID, setEmojiDeleteID] = useState("");

	const [stickerName, setStickerName] = useState("");
	const [stickerDescription, setStickerDescription] = useState("");
	const [stickerEmojiTag, setStickerEmojiTag] = useState("");
	const [stickerFile, setStickerFile] = useState<File | null>(null);
	const [stickerEditID, setStickerEditID] = useState("");
	const [stickerEditName, setStickerEditName] = useState("");
	const [stickerEditDescription, setStickerEditDescription] = useState("");
	const [stickerDeleteID, setStickerDeleteID] = useState("");

	const refreshAssets = useCallback(async () => {
		setAssetsLoading(true);
		try {
			const [channelsResp, rolesResp, membersResp, emojisResp, stickersResp] =
				await Promise.all([
					get<{ channels: GuildChannelInfo[] }>(
						`/api/guilds/channels?guild_id=${guildID}`,
					),
					get<{ roles: GuildRoleInfo[] }>(
						`/api/guilds/roles?guild_id=${guildID}`,
					),
					get<{ members: GuildMemberInfo[] }>(
						`/api/guilds/members?guild_id=${guildID}&limit=25`,
					),
					get<{ emojis: GuildEmojiInfo[] }>(
						`/api/guilds/emojis?guild_id=${guildID}`,
					),
					get<{ stickers: GuildStickerInfo[] }>(
						`/api/guilds/stickers?guild_id=${guildID}`,
					),
				]);
			setChannels(channelsResp.channels);
			setRoles(rolesResp.roles);
			setMembers(membersResp.members);
			setEmojis(emojisResp.emojis);
			setStickers(stickersResp.stickers);
		} catch (error) {
			notifyError("Could not load server resources", error);
		} finally {
			setAssetsLoading(false);
		}
	}, [guildID]);

	const searchMembers = useCallback(
		async (query: string) => {
			setMemberQuery(query);
			try {
				const response = await get<{ members: GuildMemberInfo[] }>(
					`/api/guilds/members?guild_id=${guildID}&query=${encodeURIComponent(query)}&limit=25`,
				);
				setMembers(response.members);
			} catch (error) {
				notifyError("Could not search members", error);
			}
		},
		[guildID],
	);

	const refreshWarnings = useCallback(
		async (userID: string) => {
			try {
				const response = await get<{ warnings: WarningInfo[] }>(
					`/api/guilds/moderation/warnings?guild_id=${guildID}&user_id=${encodeURIComponent(userID)}&limit=25`,
				);
				setWarnings(response.warnings);
			} catch (error) {
				notifyError("Could not load warnings", error);
			}
		},
		[guildID],
	);

	useEffect(() => {
		if (!guildDashboard) {
			return;
		}
		setFunConfig(configFromSection(guildDashboard.fun));
		setInfoConfig(configFromSection(guildDashboard.info));
		setManagerConfig(configFromSection(guildDashboard.manager));
		setModerationConfig({
			...configFromSection(guildDashboard.moderation),
			warning_limit: guildDashboard.moderation.warning_limit,
			timeout_threshold: guildDashboard.moderation.timeout_threshold,
			timeout_minutes: guildDashboard.moderation.timeout_minutes,
		});
		setWellnessConfig({
			...configFromSection(guildDashboard.wellness),
			allow_channel_reminders: guildDashboard.wellness.allow_channel_reminders,
			default_reminder_channel_id:
				guildDashboard.wellness.default_reminder_channel_id,
		});
	}, [guildDashboard]);

	useEffect(() => {
		if (!(me && guildDashboard?.guild.bot_installed)) {
			return;
		}
		refreshAssets().catch(() => undefined);
	}, [guildDashboard?.guild.bot_installed, me, refreshAssets]);

	useEffect(() => {
		if (!me || selectedMemberID === "") {
			setWarnings([]);
			return;
		}
		refreshWarnings(selectedMemberID).catch(() => undefined);
	}, [me, refreshWarnings, selectedMemberID]);

	const channelOptions = useMemo(
		() =>
			channels.map((item) => ({
				value: String(item.id),
				label: `${item.name} (${item.type})`,
			})),
		[channels],
	);
	const roleOptions = useMemo(
		() => roles.map((item) => ({ value: String(item.id), label: item.name })),
		[roles],
	);
	const memberOptions = useMemo(
		() =>
			members.map((item) => ({
				value: String(item.user_id),
				label: item.display_name || item.username,
			})),
		[members],
	);
	const emojiOptions = useMemo(
		() => emojis.map((item) => ({ value: String(item.id), label: item.name })),
		[emojis],
	);
	const stickerOptions = useMemo(
		() =>
			stickers.map((item) => ({ value: String(item.id), label: item.name })),
		[stickers],
	);

	if (!me) {
		return (
			<Stack gap="lg">
				<PageHeader
					title="Server"
					subtitle="Sign in to open this server dashboard."
				/>
				<Card className="panel-card" withBorder={true}>
					<Stack gap="sm">
						<Text fw={700}>Discord sign-in required</Text>
						<Text size="sm" c="dimmed">
							Sign in first, then come back to this server dashboard.
						</Text>
						<Group>
							<Button onClick={onLogin}>Sign in with Discord</Button>
						</Group>
					</Stack>
				</Card>
			</Stack>
		);
	}

	if (loading && !guildDashboard) {
		return (
			<Group justify="center" py="xl">
				<Loader color="goblue" />
			</Group>
		);
	}

	if (!guildDashboard) {
		return (
			<Stack gap="lg">
				<PageHeader
					title="Server"
					subtitle="This server could not be loaded."
				/>
				<Card className="panel-card" withBorder={true}>
					<Stack gap="sm">
						<Text size="sm" c="dimmed">
							The server is not available in the current session or you do not
							have access to it.
						</Text>
						<Group>
							<Button
								variant="default"
								leftSection={<IconArrowLeft size={16} />}
								onClick={onBack}
							>
								Back to servers
							</Button>
						</Group>
					</Stack>
				</Card>
			</Stack>
		);
	}

	async function savePluginConfig(pluginID: string, config: GuildPluginConfig) {
		setSavingPlugin(pluginID);
		try {
			await post(
				"/api/guilds/config",
				{
					guild_id: guildID,
					plugin_id: pluginID,
					config,
				},
				csrfToken,
			);
			notifications.show({
				color: "teal",
				title: "Saved",
				message: `${pluginID} settings updated.`,
			});
			onRefresh();
		} catch (error) {
			notifyError("Could not save settings", error);
		} finally {
			setSavingPlugin(null);
		}
	}

	async function submitWarn() {
		if (!selectedMemberID || warnReason.trim() === "") {
			return;
		}
		try {
			await post(
				"/api/guilds/moderation/warn",
				{
					guild_id: guildID,
					user_id: selectedMemberID,
					reason: warnReason,
				},
				csrfToken,
			);
			setWarnReason("");
			notifications.show({
				color: "teal",
				title: "Saved",
				message: "Warning created.",
			});
			await refreshWarnings(selectedMemberID);
			onRefresh();
		} catch (error) {
			notifyError("Could not create warning", error);
		}
	}

	async function deleteWarning(warningID: string) {
		try {
			await post(
				"/api/guilds/moderation/unwarn",
				{
					guild_id: guildID,
					warning_id: warningID,
				},
				csrfToken,
			);
			notifications.show({
				color: "teal",
				title: "Saved",
				message: "Warning removed.",
			});
			await refreshWarnings(selectedMemberID);
			onRefresh();
		} catch (error) {
			notifyError("Could not remove warning", error);
		}
	}

	async function runManagerAction(
		title: string,
		path: string,
		body: unknown,
		refresh?: () => Promise<void>,
	) {
		try {
			await post(path, body, csrfToken);
			notifications.show({ color: "teal", title: "Saved", message: title });
			if (refresh) {
				await refresh();
			}
			onRefresh();
		} catch (error) {
			notifyError(title, error);
		}
	}

	async function createEmoji() {
		if (!emojiFile || emojiName.trim() === "") {
			return;
		}
		const payload = await filePayload(emojiFile);
		await runManagerAction(
			"Emoji created.",
			"/api/guilds/manager/emojis/create",
			{
				guild_id: guildID,
				name: emojiName.trim(),
				filename: payload.filename,
				content_b64: payload.contentB64,
				width: payload.width,
				height: payload.height,
			},
			refreshAssets,
		);
		setEmojiName("");
		setEmojiFile(null);
	}

	async function createSticker() {
		if (
			!stickerFile ||
			stickerName.trim() === "" ||
			stickerEmojiTag.trim() === ""
		) {
			return;
		}
		const payload = await filePayload(stickerFile);
		await runManagerAction(
			"Sticker created.",
			"/api/guilds/manager/stickers/create",
			{
				guild_id: guildID,
				name: stickerName.trim(),
				description: stickerDescription.trim(),
				emoji_tag: stickerEmojiTag.trim(),
				filename: payload.filename,
				content_b64: payload.contentB64,
				width: payload.width,
				height: payload.height,
			},
			refreshAssets,
		);
		setStickerName("");
		setStickerDescription("");
		setStickerEmojiTag("");
		setStickerFile(null);
	}

	return (
		<Stack gap="md">
			<PageHeader
				title={guildDashboard.guild.name}
				subtitle="Server setup, feature settings, moderation, and manager actions."
				primaryAction={
					<Button
						rightSection={<IconExternalLink size={16} />}
						onClick={() => onInstall(guildID)}
					>
						{guildDashboard.guild.bot_installed ? "Reopen invite" : "Add bot"}
					</Button>
				}
				secondaryActions={[
					{
						key: "back",
						label: "Back",
						icon: <IconArrowLeft size={16} />,
						onClick: onBack,
					},
					{
						key: "refresh",
						label: "Refresh",
						icon: <IconRefresh size={16} />,
						onClick: onRefresh,
						loading,
					},
				]}
			/>

			<SimpleGrid cols={{ base: 1, md: 3 }}>
				<Card className="panel-card" withBorder={true}>
					<Stack gap="xs">
						<Group justify="space-between">
							<Text fw={700}>Server</Text>
							<CopyButton value={guildDashboard.guild.id}>
								{({ copied, copy }) => (
									<Tooltip
										label={copied ? "Copied" : "Copy server ID"}
										withArrow={true}
									>
										<ActionIcon
											variant="subtle"
											radius="md"
											aria-label="Copy server ID"
											onClick={copy}
										>
											{copied ? (
												<IconCheck size={16} />
											) : (
												<IconCopy size={16} />
											)}
										</ActionIcon>
									</Tooltip>
								)}
							</CopyButton>
						</Group>
						<Text size="sm">{guildDashboard.guild.name}</Text>
					</Stack>
				</Card>
				<Card className="panel-card" withBorder={true}>
					<Stack gap="xs">
						<Text fw={700}>Install state</Text>
						<Badge
							color={badgeColor(guildDashboard.guild.bot_installed)}
							w="fit-content"
						>
							{guildDashboard.guild.bot_installed
								? "Installed"
								: "Not installed"}
						</Badge>
						<Text size="sm" c="dimmed">
							{guildDashboard.guild.bot_installed
								? "The bot is already present in this server."
								: "Use the add-bot flow to continue."}
						</Text>
					</Stack>
				</Card>
				<Card className="panel-card" withBorder={true}>
					<Stack gap="xs">
						<Text fw={700}>Access</Text>
						<Badge
							color={badgeColor(guildDashboard.guild.can_manage)}
							w="fit-content"
						>
							{guildDashboard.guild.owner ? "Owner" : "Manager"}
						</Badge>
						<Text size="sm" c="dimmed">
							Your Discord account can manage this server.
						</Text>
					</Stack>
				</Card>
			</SimpleGrid>

			<SimpleGrid cols={{ base: 1, md: 2 }} spacing="md">
				{guildDashboard.setup_checks.map((check) => (
					<Card key={check.id} className="panel-card" withBorder={true}>
						<Stack gap="xs">
							<Group justify="space-between">
								<Text fw={700}>{check.label}</Text>
								<Badge color={badgeColor(check.ok)}>
									{check.ok ? "OK" : "Action needed"}
								</Badge>
							</Group>
							<Text size="sm" c="dimmed">
								{check.message}
							</Text>
						</Stack>
					</Card>
				))}
			</SimpleGrid>

			<Tabs defaultValue="manager" variant="outline">
				<Tabs.List>
					<Tabs.Tab value="manager">Manager</Tabs.Tab>
					<Tabs.Tab value="moderation">Moderation</Tabs.Tab>
					<Tabs.Tab value="fun">Fun</Tabs.Tab>
					<Tabs.Tab value="info">Info</Tabs.Tab>
					<Tabs.Tab value="wellness">Wellness</Tabs.Tab>
				</Tabs.List>

				<Tabs.Panel value="manager" pt="md">
					<Stack gap="md">
						<PluginSettingsCard
							title="Manager settings"
							section={guildDashboard.manager}
							config={managerConfig}
							saving={savingPlugin === "manager"}
							onToggleEnabled={(enabled) =>
								setManagerConfig((current) => ({ ...current, enabled }))
							}
							onToggleCommand={(command, enabled) =>
								setManagerConfig((current) => ({
									...current,
									commands: { ...current.commands, [command]: enabled },
								}))
							}
							onSave={() => savePluginConfig("manager", managerConfig)}
						/>

						<SimpleGrid cols={{ base: 1, md: 4 }}>
							<MetricCard
								label="Channels"
								value={String(guildDashboard.manager.channel_count)}
							/>
							<MetricCard
								label="Roles"
								value={String(guildDashboard.manager.role_count)}
							/>
							<MetricCard
								label="Emojis"
								value={String(guildDashboard.manager.emoji_count)}
							/>
							<MetricCard
								label="Stickers"
								value={String(guildDashboard.manager.sticker_count)}
							/>
						</SimpleGrid>

						<Accordion variant="contained">
							<Accordion.Item value="slowmode">
								<Accordion.Control>Slowmode</Accordion.Control>
								<Accordion.Panel>
									<Stack>
										<Select
											label="Channel"
											data={channelOptions}
											value={slowmodeChannelID}
											onChange={(value) => setSlowmodeChannelID(value ?? "")}
										/>
										<NumberInput
											label="Seconds"
											min={0}
											value={slowmodeSeconds}
											onChange={(value) =>
												setSlowmodeSeconds(
													typeof value === "number" ? value : "",
												)
											}
										/>
										<Group>
											<Button
												onClick={() =>
													runManagerAction(
														"Slowmode updated.",
														"/api/guilds/manager/slowmode",
														{
															guild_id: guildID,
															channel_id: slowmodeChannelID,
															seconds: Number(slowmodeSeconds) || 0,
														},
													)
												}
											>
												Save slowmode
											</Button>
										</Group>
									</Stack>
								</Accordion.Panel>
							</Accordion.Item>

							<Accordion.Item value="nickname">
								<Accordion.Control>Nickname</Accordion.Control>
								<Accordion.Panel>
									<Stack>
										<Group grow={true}>
											<TextInput
												label="Member search"
												value={memberQuery}
												onChange={(event) =>
													setMemberQuery(event.currentTarget.value)
												}
											/>
											<Button
												variant="default"
												onClick={() => {
													searchMembers(memberQuery).catch(() => undefined);
												}}
											>
												Search
											</Button>
										</Group>
										<Select
											label="Member"
											data={memberOptions}
											value={nicknameMemberID}
											onChange={(value) => setNicknameMemberID(value ?? "")}
										/>
										<TextInput
											label="Nickname"
											description="Leave empty to reset the nickname."
											value={nicknameValue}
											onChange={(event) =>
												setNicknameValue(event.currentTarget.value)
											}
										/>
										<Button
											onClick={() =>
												runManagerAction(
													"Nickname updated.",
													"/api/guilds/manager/nick",
													{
														guild_id: guildID,
														user_id: nicknameMemberID,
														nickname: nicknameValue,
													},
												)
											}
										>
											Save nickname
										</Button>
									</Stack>
								</Accordion.Panel>
							</Accordion.Item>

							<Accordion.Item value="roles">
								<Accordion.Control>Roles</Accordion.Control>
								<Accordion.Panel>
									<Stack>
										<SimpleGrid cols={{ base: 1, md: 2 }}>
											<Card className="nested-panel" withBorder={true}>
												<Stack>
													<Text fw={700}>Create role</Text>
													<TextInput
														label="Name"
														value={roleName}
														onChange={(event) =>
															setRoleName(event.currentTarget.value)
														}
													/>
													<TextInput
														label="Color"
														placeholder="#5865F2"
														value={roleColor}
														onChange={(event) =>
															setRoleColor(event.currentTarget.value)
														}
													/>
													<Button
														onClick={() =>
															runManagerAction(
																"Role created.",
																"/api/guilds/manager/roles/create",
																{
																	guild_id: guildID,
																	name: roleName,
																	color: parseHexColor(roleColor),
																},
																refreshAssets,
															)
														}
													>
														Create role
													</Button>
												</Stack>
											</Card>

											<Card className="nested-panel" withBorder={true}>
												<Stack>
													<Text fw={700}>Edit or delete role</Text>
													<Select
														label="Role"
														data={roleOptions}
														value={roleEditID}
														onChange={(value) => setRoleEditID(value ?? "")}
													/>
													<TextInput
														label="New name"
														value={roleName}
														onChange={(event) =>
															setRoleName(event.currentTarget.value)
														}
													/>
													<TextInput
														label="New color"
														placeholder="#5865F2"
														value={roleColor}
														onChange={(event) =>
															setRoleColor(event.currentTarget.value)
														}
													/>
													<Group grow={true}>
														<Button
															onClick={() =>
																runManagerAction(
																	"Role updated.",
																	"/api/guilds/manager/roles/edit",
																	{
																		guild_id: guildID,
																		role_id: roleEditID,
																		name: roleName,
																		color: parseHexColor(roleColor),
																	},
																	refreshAssets,
																)
															}
														>
															Save role
														</Button>
														<Button
															color="red"
															variant="light"
															onClick={() =>
																runManagerAction(
																	"Role deleted.",
																	"/api/guilds/manager/roles/delete",
																	{
																		guild_id: guildID,
																		role_id: roleEditID,
																	},
																	refreshAssets,
																)
															}
														>
															Delete
														</Button>
													</Group>
												</Stack>
											</Card>
										</SimpleGrid>

										<Card className="nested-panel" withBorder={true}>
											<Stack>
												<Text fw={700}>Assign or remove role</Text>
												<Group grow={true}>
													<Select
														label="Member"
														data={memberOptions}
														value={roleMemberID}
														onChange={(value) => setRoleMemberID(value ?? "")}
													/>
													<Select
														label="Role"
														data={roleOptions}
														value={roleMemberRoleID}
														onChange={(value) =>
															setRoleMemberRoleID(value ?? "")
														}
													/>
													<Select
														label="Action"
														data={[
															{ value: "add", label: "Add" },
															{ value: "remove", label: "Remove" },
														]}
														value={roleMode}
														onChange={(value) =>
															setRoleMode((value as "add" | "remove") ?? "add")
														}
													/>
												</Group>
												<Button
													onClick={() =>
														runManagerAction(
															"Role membership updated.",
															"/api/guilds/manager/roles/member",
															{
																add: roleMode === "add",
																guild_id: guildID,
																user_id: roleMemberID,
																role_id: roleMemberRoleID,
															},
														)
													}
												>
													Save member role
												</Button>
											</Stack>
										</Card>
									</Stack>
								</Accordion.Panel>
							</Accordion.Item>

							<Accordion.Item value="purge">
								<Accordion.Control>Purge</Accordion.Control>
								<Accordion.Panel>
									<Stack>
										<Select
											label="Channel"
											data={channelOptions}
											value={purgeChannelID}
											onChange={(value) => setPurgeChannelID(value ?? "")}
										/>
										<Group grow={true}>
											<Select
												label="Mode"
												data={[
													{ value: "all", label: "All" },
													{ value: "before", label: "Before message" },
													{ value: "after", label: "After message" },
													{ value: "around", label: "Around message" },
												]}
												value={purgeMode}
												onChange={(value) => setPurgeMode(value ?? "all")}
											/>
											<NumberInput
												label="Count"
												min={1}
												max={100}
												value={purgeCount}
												onChange={(value) =>
													setPurgeCount(typeof value === "number" ? value : "")
												}
											/>
										</Group>
										<TextInput
											label="Anchor message ID"
											description="Required for before, after, and around modes."
											value={purgeAnchor}
											onChange={(event) =>
												setPurgeAnchor(event.currentTarget.value)
											}
										/>
										<Button
											color="red"
											onClick={() =>
												runManagerAction(
													"Messages purged.",
													"/api/guilds/manager/purge",
													{
														guild_id: guildID,
														channel_id: purgeChannelID,
														mode: purgeMode,
														anchor_raw: purgeAnchor,
														count: Number(purgeCount) || 1,
													},
												)
											}
										>
											Run purge
										</Button>
									</Stack>
								</Accordion.Panel>
							</Accordion.Item>

							<Accordion.Item value="emojis">
								<Accordion.Control>Emojis</Accordion.Control>
								<Accordion.Panel>
									<Stack>
										<Card className="nested-panel" withBorder={true}>
											<Stack>
												<Text fw={700}>Create emoji</Text>
												<TextInput
													label="Name"
													value={emojiName}
													onChange={(event) =>
														setEmojiName(event.currentTarget.value)
													}
												/>
												<FileInput
													label="Image file"
													value={emojiFile}
													onChange={setEmojiFile}
												/>
												<Button
													onClick={() => createEmoji().catch(() => undefined)}
												>
													Create emoji
												</Button>
											</Stack>
										</Card>
										<Group grow={true}>
											<Card className="nested-panel" withBorder={true}>
												<Stack>
													<Text fw={700}>Rename emoji</Text>
													<Select
														label="Emoji"
														data={emojiOptions}
														value={emojiEditID}
														onChange={(value) => setEmojiEditID(value ?? "")}
													/>
													<TextInput
														label="New name"
														value={emojiEditName}
														onChange={(event) =>
															setEmojiEditName(event.currentTarget.value)
														}
													/>
													<Button
														onClick={() =>
															runManagerAction(
																"Emoji updated.",
																"/api/guilds/manager/emojis/edit",
																{
																	guild_id: guildID,
																	raw_emoji: emojiEditID,
																	name: emojiEditName,
																},
																refreshAssets,
															)
														}
													>
														Save emoji
													</Button>
												</Stack>
											</Card>
											<Card className="nested-panel" withBorder={true}>
												<Stack>
													<Text fw={700}>Delete emoji</Text>
													<Select
														label="Emoji"
														data={emojiOptions}
														value={emojiDeleteID}
														onChange={(value) => setEmojiDeleteID(value ?? "")}
													/>
													<Button
														color="red"
														variant="light"
														onClick={() =>
															runManagerAction(
																"Emoji deleted.",
																"/api/guilds/manager/emojis/delete",
																{
																	guild_id: guildID,
																	raw_emoji: emojiDeleteID,
																},
																refreshAssets,
															)
														}
													>
														Delete emoji
													</Button>
												</Stack>
											</Card>
										</Group>
									</Stack>
								</Accordion.Panel>
							</Accordion.Item>

							<Accordion.Item value="stickers">
								<Accordion.Control>Stickers</Accordion.Control>
								<Accordion.Panel>
									<Stack>
										<Card className="nested-panel" withBorder={true}>
											<Stack>
												<Text fw={700}>Create sticker</Text>
												<TextInput
													label="Name"
													value={stickerName}
													onChange={(event) =>
														setStickerName(event.currentTarget.value)
													}
												/>
												<TextInput
													label="Emoji tag"
													value={stickerEmojiTag}
													onChange={(event) =>
														setStickerEmojiTag(event.currentTarget.value)
													}
												/>
												<TextInput
													label="Description"
													value={stickerDescription}
													onChange={(event) =>
														setStickerDescription(event.currentTarget.value)
													}
												/>
												<FileInput
													label="Sticker file"
													value={stickerFile}
													onChange={setStickerFile}
												/>
												<Button
													onClick={() => createSticker().catch(() => undefined)}
												>
													Create sticker
												</Button>
											</Stack>
										</Card>
										<Group grow={true}>
											<Card className="nested-panel" withBorder={true}>
												<Stack>
													<Text fw={700}>Edit sticker</Text>
													<Select
														label="Sticker"
														data={stickerOptions}
														value={stickerEditID}
														onChange={(value) => setStickerEditID(value ?? "")}
													/>
													<TextInput
														label="New name"
														value={stickerEditName}
														onChange={(event) =>
															setStickerEditName(event.currentTarget.value)
														}
													/>
													<TextInput
														label="New description"
														value={stickerEditDescription}
														onChange={(event) =>
															setStickerEditDescription(
																event.currentTarget.value,
															)
														}
													/>
													<Button
														onClick={() =>
															runManagerAction(
																"Sticker updated.",
																"/api/guilds/manager/stickers/edit",
																{
																	guild_id: guildID,
																	raw_id: stickerEditID,
																	name: stickerEditName,
																	description: stickerEditDescription,
																},
																refreshAssets,
															)
														}
													>
														Save sticker
													</Button>
												</Stack>
											</Card>
											<Card className="nested-panel" withBorder={true}>
												<Stack>
													<Text fw={700}>Delete sticker</Text>
													<Select
														label="Sticker"
														data={stickerOptions}
														value={stickerDeleteID}
														onChange={(value) =>
															setStickerDeleteID(value ?? "")
														}
													/>
													<Button
														color="red"
														variant="light"
														onClick={() =>
															runManagerAction(
																"Sticker deleted.",
																"/api/guilds/manager/stickers/delete",
																{
																	guild_id: guildID,
																	raw_id: stickerDeleteID,
																},
																refreshAssets,
															)
														}
													>
														Delete sticker
													</Button>
												</Stack>
											</Card>
										</Group>
									</Stack>
								</Accordion.Panel>
							</Accordion.Item>
						</Accordion>
					</Stack>
				</Tabs.Panel>

				<Tabs.Panel value="moderation" pt="md">
					<Stack gap="md">
						<PluginSettingsCard
							title="Moderation settings"
							section={guildDashboard.moderation}
							config={moderationConfig}
							saving={savingPlugin === "moderation"}
							onToggleEnabled={(enabled) =>
								setModerationConfig((current) => ({ ...current, enabled }))
							}
							onToggleCommand={(command, enabled) =>
								setModerationConfig((current) => ({
									...current,
									commands: { ...current.commands, [command]: enabled },
								}))
							}
							extra={
								<SimpleGrid cols={{ base: 1, md: 3 }}>
									<NumberInput
										label="Warning limit"
										min={1}
										max={20}
										value={moderationConfig.warning_limit ?? 3}
										onChange={(value) =>
											setModerationConfig((current) => ({
												...current,
												warning_limit: numberValue(value, 3),
											}))
										}
									/>
									<NumberInput
										label="Timeout threshold"
										min={1}
										max={20}
										value={moderationConfig.timeout_threshold ?? 3}
										onChange={(value) =>
											setModerationConfig((current) => ({
												...current,
												timeout_threshold: numberValue(value, 3),
											}))
										}
									/>
									<NumberInput
										label="Timeout minutes"
										min={1}
										max={10080}
										value={moderationConfig.timeout_minutes ?? 10}
										onChange={(value) =>
											setModerationConfig((current) => ({
												...current,
												timeout_minutes: numberValue(value, 10),
											}))
										}
									/>
								</SimpleGrid>
							}
							onSave={() => savePluginConfig("moderation", moderationConfig)}
						/>

						<Card className="panel-card" withBorder={true}>
							<Stack gap="md">
								<Text fw={700}>Members</Text>
								<Group grow={true}>
									<TextInput
										label="Search members"
										value={memberQuery}
										onChange={(event) =>
											setMemberQuery(event.currentTarget.value)
										}
									/>
									<Button
										variant="default"
										onClick={() => {
											searchMembers(memberQuery).catch(() => undefined);
										}}
									>
										Search
									</Button>
								</Group>
								<Select
									label="Member"
									data={memberOptions}
									value={selectedMemberID}
									onChange={(value) => setSelectedMemberID(value ?? "")}
								/>
								<Textarea
									label="Warn reason"
									minRows={2}
									value={warnReason}
									onChange={(event) => setWarnReason(event.currentTarget.value)}
								/>
								<Group>
									<Button onClick={() => submitWarn().catch(() => undefined)}>
										Warn member
									</Button>
								</Group>
								<Divider />
								<Text fw={700}>Warnings</Text>
								{warnings.length === 0 ? (
									<Text size="sm" c="dimmed">
										No warnings loaded for the selected member.
									</Text>
								) : (
									<Table
										className="compact-table"
										striped={true}
										highlightOnHover={true}
									>
										<Table.Thead>
											<Table.Tr>
												<Table.Th>When</Table.Th>
												<Table.Th>Moderator</Table.Th>
												<Table.Th>Reason</Table.Th>
												<Table.Th />
											</Table.Tr>
										</Table.Thead>
										<Table.Tbody>
											{warnings.map((warning) => (
												<Table.Tr key={warning.id}>
													<Table.Td>
														{new Date(warning.created_at).toLocaleString()}
													</Table.Td>
													<Table.Td>{warning.moderator_id}</Table.Td>
													<Table.Td>{warning.reason}</Table.Td>
													<Table.Td>
														<Button
															variant="light"
															color="red"
															size="xs"
															onClick={() => {
																deleteWarning(warning.id).catch(
																	() => undefined,
																);
															}}
														>
															Remove
														</Button>
													</Table.Td>
												</Table.Tr>
											))}
										</Table.Tbody>
									</Table>
								)}
							</Stack>
						</Card>
					</Stack>
				</Tabs.Panel>

				<Tabs.Panel value="fun" pt="md">
					<PluginSettingsCard
						title="Fun settings"
						section={guildDashboard.fun}
						config={funConfig}
						saving={savingPlugin === "fun"}
						onToggleEnabled={(enabled) =>
							setFunConfig((current) => ({ ...current, enabled }))
						}
						onToggleCommand={(command, enabled) =>
							setFunConfig((current) => ({
								...current,
								commands: { ...current.commands, [command]: enabled },
							}))
						}
						onSave={() => savePluginConfig("fun", funConfig)}
					/>
				</Tabs.Panel>

				<Tabs.Panel value="info" pt="md">
					<PluginSettingsCard
						title="Info settings"
						section={guildDashboard.info}
						config={infoConfig}
						saving={savingPlugin === "info"}
						onToggleEnabled={(enabled) =>
							setInfoConfig((current) => ({ ...current, enabled }))
						}
						onToggleCommand={(command, enabled) =>
							setInfoConfig((current) => ({
								...current,
								commands: { ...current.commands, [command]: enabled },
							}))
						}
						onSave={() => savePluginConfig("info", infoConfig)}
					/>
				</Tabs.Panel>

				<Tabs.Panel value="wellness" pt="md">
					<PluginSettingsCard
						title="Wellness settings"
						section={guildDashboard.wellness}
						config={wellnessConfig}
						saving={savingPlugin === "wellness"}
						onToggleEnabled={(enabled) =>
							setWellnessConfig((current) => ({ ...current, enabled }))
						}
						onToggleCommand={(command, enabled) =>
							setWellnessConfig((current) => ({
								...current,
								commands: { ...current.commands, [command]: enabled },
							}))
						}
						extra={
							<SimpleGrid cols={{ base: 1, md: 2 }}>
								<Switch
									label="Allow channel reminders"
									checked={wellnessConfig.allow_channel_reminders === true}
									onChange={(event) =>
										setWellnessConfig((current) => ({
											...current,
											allow_channel_reminders: event.currentTarget.checked,
										}))
									}
								/>
								<Select
									label="Default reminder channel"
									clearable={true}
									data={channelOptions}
									value={wellnessConfig.default_reminder_channel_id ?? null}
									onChange={(value) =>
										setWellnessConfig((current) => ({
											...current,
											default_reminder_channel_id: value ?? undefined,
										}))
									}
								/>
							</SimpleGrid>
						}
						onSave={() => savePluginConfig("wellness", wellnessConfig)}
					/>
				</Tabs.Panel>
			</Tabs>

			{assetsLoading ? (
				<Group justify="center" py="sm">
					<Loader size="sm" />
				</Group>
			) : null}
		</Stack>
	);
}

function PluginSettingsCard({
	title,
	section,
	config,
	saving,
	onToggleEnabled,
	onToggleCommand,
	onSave,
	extra,
}: {
	title: string;
	section: PluginSection;
	config: GuildPluginConfig;
	saving: boolean;
	onToggleEnabled: (enabled: boolean) => void;
	onToggleCommand: (command: string, enabled: boolean) => void;
	onSave: () => void;
	extra?: ReactNode;
}) {
	return (
		<Card className="panel-card" withBorder={true}>
			<Stack gap="md">
				<Group justify="space-between" align="flex-start">
					<Stack gap={4}>
						<Text fw={700}>{title}</Text>
						<Group gap="xs">
							<Badge color={badgeColor(section.global_enabled)}>
								{section.global_enabled ? "Global on" : "Global off"}
							</Badge>
							<Badge color={badgeColor(config.enabled)}>
								Server {config.enabled ? "on" : "off"}
							</Badge>
						</Group>
					</Stack>
					<Switch
						checked={config.enabled}
						onChange={(event) => onToggleEnabled(event.currentTarget.checked)}
					/>
				</Group>
				<SimpleGrid cols={{ base: 1, md: 2 }}>
					{section.commands.map((command) => (
						<Switch
							key={command.id}
							label={command.label}
							checked={config.commands[command.id] !== false}
							onChange={(event) =>
								onToggleCommand(command.id, event.currentTarget.checked)
							}
						/>
					))}
				</SimpleGrid>
				{extra}
				<Group justify="flex-end">
					<Button
						loading={saving}
						disabled={!section.global_enabled}
						onClick={onSave}
					>
						Save settings
					</Button>
				</Group>
			</Stack>
		</Card>
	);
}

function configFromSection(section: PluginSection): GuildPluginConfig {
	const commands: Record<string, boolean> = {};
	for (const command of section.commands) {
		commands[command.id] = command.enabled;
	}
	return {
		enabled: section.enabled,
		commands,
	};
}

function emptyConfig(): GuildPluginConfig {
	return {
		enabled: true,
		commands: {},
	};
}

function emptyModerationConfig(): GuildPluginConfig {
	return {
		...emptyConfig(),
		warning_limit: 3,
		timeout_threshold: 3,
		timeout_minutes: 10,
	};
}

function emptyWellnessConfig(): GuildPluginConfig {
	return {
		...emptyConfig(),
		allow_channel_reminders: true,
	};
}

function numberValue(value: string | number, fallback: number): number {
	if (typeof value === "number" && Number.isFinite(value)) {
		return value;
	}
	const parsed = Number(value);
	if (Number.isFinite(parsed)) {
		return parsed;
	}
	return fallback;
}

function parseHexColor(raw: string): number | undefined {
	const trimmed = raw.trim().replace(LEADING_HASH_RE, "");
	if (trimmed.length !== 6 || NON_HEX_RE.test(trimmed)) {
		return undefined;
	}
	return Number.parseInt(trimmed, 16);
}

async function filePayload(file: File): Promise<{
	filename: string;
	contentB64: string;
	width: number;
	height: number;
}> {
	const contentB64 = await new Promise<string>((resolve, reject) => {
		const reader = new FileReader();
		reader.onerror = () => reject(new Error("Could not read file."));
		reader.onload = () => resolve(String(reader.result ?? ""));
		reader.readAsDataURL(file);
	});

	const dimensions = await new Promise<{ width: number; height: number }>(
		(resolve) => {
			const image = new Image();
			image.onload = () =>
				resolve({ width: image.naturalWidth, height: image.naturalHeight });
			image.onerror = () => resolve({ width: 0, height: 0 });
			image.src = URL.createObjectURL(file);
		},
	);

	return {
		filename: file.name,
		contentB64,
		width: dimensions.width,
		height: dimensions.height,
	};
}

function notifyError(title: string, error: unknown) {
	notifications.show({
		color: "red",
		title,
		message: error instanceof Error ? error.message : "Unknown error",
	});
}
