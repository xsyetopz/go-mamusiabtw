import {
	Badge,
	Button,
	Card,
	Code,
	Group,
	SimpleGrid,
	Stack,
	Text,
} from "@mantine/core";
import { IconRefresh } from "@tabler/icons-react";
import { PageHeader } from "../components/PageHeader";
import { CodeLine, SetupMessage, StatePanel } from "../components/StatePanel";
import { type BootstrapState, localSetup } from "../dashboard";
import { badgeColor } from "../format";
import type { SetupStatus, StatusResponse } from "../types";

type Props = {
	bootstrap: BootstrapState;
	setupStatus: SetupStatus | null;
	status: StatusResponse | null;
	apiBase: string;
	onRefresh: () => void;
	onLogin: () => void;
};

function ConnectionPanel({
	bootstrap,
	apiBase,
	resolvedAdminAddr,
}: {
	bootstrap: BootstrapState;
	apiBase: string;
	resolvedAdminAddr: string;
}) {
	return (
		<StatePanel
			title="Connection"
			status={
				<Badge
					color={badgeColor(
						bootstrap.kind !== "offline" &&
							bootstrap.kind !== "invalid_api_base",
					)}
				>
					{bootstrap.kind === "offline" || bootstrap.kind === "invalid_api_base"
						? "Needs attention"
						: "Available"}
				</Badge>
			}
		>
			<CodeLine label="Admin API URL" value={apiBase} />
			<CodeLine label="Expected local admin API" value={resolvedAdminAddr} />
			{bootstrap.kind === "invalid_api_base" ? (
				<SetupMessage title="Invalid dashboard URL">
					{bootstrap.message}
				</SetupMessage>
			) : null}
			{bootstrap.kind === "offline" ? (
				<SetupMessage title="Admin API not reachable">
					{bootstrap.message}
				</SetupMessage>
			) : null}
		</StatePanel>
	);
}

function SignInPanel({
	setupStatus,
	resolvedOrigin,
	resolvedRedirect,
	bootstrap,
	onLogin,
}: {
	setupStatus: SetupStatus | null;
	resolvedOrigin: string;
	resolvedRedirect: string;
	bootstrap: BootstrapState;
	onLogin: () => void;
}) {
	return (
		<StatePanel
			title="Sign-in"
			status={
				<Badge color={badgeColor(setupStatus?.login_ready ?? false)}>
					{setupStatus?.login_ready ? "Ready" : "Not ready"}
				</Badge>
			}
		>
			<CodeLine label="Dashboard origin" value={resolvedOrigin} />
			<CodeLine label="Callback URL" value={resolvedRedirect} />
			<Stack gap="xs">
				<Badge color={badgeColor(setupStatus?.has_client_id ?? false)}>
					Client ID
				</Badge>
				<Badge color={badgeColor(setupStatus?.has_client_secret ?? false)}>
					Client secret
				</Badge>
				<Badge color={badgeColor(setupStatus?.has_session_secret ?? false)}>
					Session secret
				</Badge>
				<Badge color={badgeColor(setupStatus?.owner_resolved ?? false)}>
					Owner account
				</Badge>
			</Stack>
			<CodeLine
				label="Owner source"
				value={setupStatus?.owner_source || "unresolved"}
			/>
			{setupStatus?.effective_owner_user_id ? (
				<CodeLine
					label="Effective owner user ID"
					value={String(setupStatus.effective_owner_user_id)}
				/>
			) : null}
			{bootstrap.kind === "unauthenticated" ? (
				<Button onClick={onLogin}>Sign in with Discord</Button>
			) : null}
		</StatePanel>
	);
}

function RuntimePanel({
	setupStatus,
	status,
}: {
	setupStatus: SetupStatus | null;
	status: StatusResponse | null;
}) {
	return (
		<StatePanel
			title="Runtime"
			status={
				<Badge color={badgeColor(setupStatus?.signing_configured ?? false)}>
					{setupStatus?.signing_configured
						? "Signing ready"
						: "Signing optional"}
				</Badge>
			}
		>
			<Group gap="xs" align="center">
				<Text size="sm">Trusted keys</Text>
				<Badge
					color={badgeColor(setupStatus?.trusted_keys_configured ?? false)}
				>
					{setupStatus?.trusted_keys_configured
						? "Configured"
						: "Not configured"}
				</Badge>
			</Group>
			{status ? (
				<>
					<CodeLine label="Plugins path" value={status.config.plugins_dir} />
					<CodeLine label="Modules file" value={status.config.modules_file} />
					<CodeLine
						label="Permissions file"
						value={status.config.permissions_file}
					/>
					<CodeLine
						label="Trusted keys file"
						value={status.config.trusted_keys_file}
					/>
				</>
			) : null}
		</StatePanel>
	);
}

function LocalSetupCard({ setupStatus }: { setupStatus: SetupStatus | null }) {
	return (
		<Card className="panel-card" withBorder={true}>
			<Stack gap="sm">
				<Text fw={700}>Local setup</Text>
				<Text size="sm">Use these values for a normal local run:</Text>
				<Code block={true}>{`MAMUSIABTW_ADMIN_ADDR=${localSetup.adminAddr}
MAMUSIABTW_DASHBOARD_APP_ORIGIN=${localSetup.appOrigin}
MAMUSIABTW_DASHBOARD_REDIRECT_URL=${localSetup.redirectURL}`}</Code>
				{(setupStatus?.hints.length ?? 0) > 0 ? (
					<Stack gap="xs">
						<Text size="sm" fw={600}>
							What still needs attention
						</Text>
						{setupStatus?.hints.map((hint) => (
							<Text key={hint} size="sm">
								{hint}
							</Text>
						))}
					</Stack>
				) : (
					<Text size="sm">The dashboard setup looks complete.</Text>
				)}
			</Stack>
		</Card>
	);
}

export function SetupPage({
	bootstrap,
	setupStatus,
	status,
	apiBase,
	onRefresh,
	onLogin,
}: Props) {
	const resolvedAdminAddr = setupStatus?.admin_addr || localSetup.adminAddr;
	const resolvedOrigin = setupStatus?.app_origin || localSetup.appOrigin;
	const resolvedRedirect = setupStatus?.redirect_url || localSetup.redirectURL;

	return (
		<Stack gap="lg">
			<PageHeader
				title="Setup"
				subtitle="Connection, sign-in, and local runtime checks."
				action={
					<Button
						variant="default"
						leftSection={<IconRefresh size={16} />}
						onClick={onRefresh}
					>
						Refresh
					</Button>
				}
			/>
			<SimpleGrid cols={{ base: 1, md: 3 }} spacing="md">
				<ConnectionPanel
					bootstrap={bootstrap}
					apiBase={apiBase}
					resolvedAdminAddr={resolvedAdminAddr}
				/>
				<SignInPanel
					setupStatus={setupStatus}
					resolvedOrigin={resolvedOrigin}
					resolvedRedirect={resolvedRedirect}
					bootstrap={bootstrap}
					onLogin={onLogin}
				/>
				<RuntimePanel setupStatus={setupStatus} status={status} />
			</SimpleGrid>
			<LocalSetupCard setupStatus={setupStatus} />
		</Stack>
	);
}
