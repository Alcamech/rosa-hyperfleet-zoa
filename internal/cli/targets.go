package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"

	"github.com/openshift-online/rosa-seaplane-zoa/internal/client"
	"github.com/openshift-online/rosa-seaplane-zoa/internal/output"
)

func newTargetsCommand(opts *GlobalOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "targets <deployment>",
		Short: "List targets within a deployment",
		Long: `List the targets (RC and MC clusters) available in a given deployment.

Resolves the deployment from Central SSM (us-east-1, then eu-west-1 on error,
or --deployments-region only), assumes the invoker role, and calls the Access
Lambda. Target rows come from the regional account (not from your laptop SSM).

Use the TARGET column with zoa session start (e.g. regional, mc01).
STATUS is Terraform registration metadata (ready = eligible for session start).

Optional: --deployments-region to pin Central SSM; --access-url and
--invoker-role-arn together to skip Central deployment SSM autodiscovery.`,
		Example: `  zoa targets us-east-1

  zoa targets us-east-1 -o wide
  zoa targets us-east-1 -o json

  zoa targets us-east-1 --deployments-region eu-west-1

  zoa targets us-east-1 \
    --access-url 'https://<id>.lambda-url.us-east-1.on.aws/' \
    --invoker-role-arn 'arn:aws:iam::123456789012:role/us-east-1-zoa-access-invoker'`,
		Args: func(_ *cobra.Command, args []string) error {
			if len(args) < 1 {
				return fmt.Errorf("deployment is required: zoa targets <deployment>\n\n  Run 'zoa deployments' to list deployment names")
			}
			if len(args) > 1 {
				return fmt.Errorf("too many arguments: expected one deployment name (got %d)", len(args))
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			deployment := args[0]
			if opts.APIURL == "" {
				opts.Deployment = deployment
			}

			c, err := getClient(opts)
			if err != nil {
				return fmt.Errorf("creating client: %w", err)
			}

			list, err := c.ListTargets(cmd.Context())
			if err != nil {
				return fmt.Errorf("listing targets: %w", err)
			}

			if opts.OutputFormat == output.FormatJSON {
				enc := json.NewEncoder(os.Stdout)
				enc.SetIndent("", "  ")
				return enc.Encode(list)
			}

			if len(list.Items) == 0 {
				fmt.Printf("No targets found for deployment %q\n", deployment)
				return nil
			}

			return printTargetsTable(os.Stdout, opts, list.Items)
		},
	}

	return cmd
}

func printTargetsTable(w io.Writer, opts *GlobalOptions, items []client.Target) error {
	wide := opts.OutputFormat == output.FormatWide
	tw := output.NewTable(w)
	if wide {
		fmt.Fprintln(tw, "TARGET\tDEPLOYMENT\tREGION\tTYPE\tSTATUS\tACCOUNT\tVPC\tSUBNETS\tSECURITY GROUP\tECS CLUSTER\tTASK DEFINITION\tFUNCTION URL")
	} else {
		fmt.Fprintln(tw, "TARGET\tREGION\tTYPE\tSTATUS")
	}
	for _, t := range items {
		if wide {
			fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
				t.TargetID,
				output.Dash(t.DeploymentName),
				output.Dash(t.Region),
				t.TargetType,
				output.Dash(t.Status),
				output.Dash(t.AccountId),
				output.Dash(t.VpcId),
				output.Dash(t.SubnetIds),
				output.Dash(t.SecurityGroupId),
				output.Dash(t.EcsClusterArn),
				output.Dash(t.TaskDefinitionArn),
				output.Dash(t.FunctionUrl))
			continue
		}
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n",
			t.TargetID, output.Dash(t.Region), t.TargetType, output.Dash(t.Status))
	}
	return tw.Flush()
}
