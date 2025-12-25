import kopf
import kubernetes.client
from kubernetes.client.rest import ApiException
import logging

# Configure logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

# Constants
ARGOCD_GROUP = "argoproj.io"
ARGOCD_VERSION = "v1alpha1"
ARGOCD_PLURAL = "applications"
LABEL_SELECTOR = "argocd.argoproj.io/managed-revision=true"
ANNOTATION_APP_NAME = "argocd.argoproj.io/application-name"
KEY_REVISION = "TARGET_REVISION"

@kopf.on.startup()
def configure(settings: kopf.OperatorSettings, **_):
    # Disable peering to run as a single instance
    settings.peering.standalone = True

def is_revision_secret(meta, **_):
    name = meta.get('name', '')
    namespace = meta.get('namespace', '')
    return namespace == 'argocd' and name.endswith('-revision')

@kopf.on.create('secrets', when=is_revision_secret)
@kopf.on.update('secrets', when=is_revision_secret)
def handle_secret_change(body, meta, spec, status, **kwargs):
    namespace = meta.get('namespace')
    secret_name = meta.get('name')
    
    # Deduce application name from secret name (remove -revision suffix)
    app_name = secret_name.replace('-revision', '')
    
    logger.info(f"Processing revision secret for app: {app_name}")

    # Extract revision from secret data
    data = body.get('data', {})
    import base64
    encoded_revision = data.get(KEY_REVISION)
    
    if not encoded_revision:
        logger.warning(f"Secret {namespace}/{secret_name} is missing key {KEY_REVISION}")
        return
        
    new_revision = base64.b64decode(encoded_revision).decode('utf-8').strip()
    logger.info(f"Detected target revision change for {app_name}: {new_revision}")

    # Patch ArgoCD Application
    custom_objects_api = kubernetes.client.CustomObjectsApi()
    
    try:
        # We assume the Application is in the 'argocd' namespace
        # or we could make this configurable via annotation too.
        argocd_namespace = annotations.get("argocd.argoproj.io/application-namespace", "argocd")
        
        # Read current application to avoid unnecessary patches
        app = custom_objects_api.get_namespaced_custom_object(
            group=ARGOCD_GROUP,
            version=ARGOCD_VERSION,
            namespace=argocd_namespace,
            plural=ARGOCD_PLURAL,
            name=app_name
        )
        
        current_revision = app.get('spec', {}).get('source', {}).get('targetRevision')
        
        if current_revision == new_revision:
            logger.info(f"Application {app_name} already at revision {new_revision}. No patch needed.")
            return

        # Prepare patch
        patch = {
            "spec": {
                "source": {
                    "targetRevision": new_revision
                }
            }
        }

        custom_objects_api.patch_namespaced_custom_object(
            group=ARGOCD_GROUP,
            version=ARGOCD_VERSION,
            namespace=argocd_namespace,
            plural=ARGOCD_PLURAL,
            name=app_name,
            body=patch
        )
        logger.info(f"Successfully patched Application {app_name} to revision {new_revision}")

    except ApiException as e:
        logger.error(f"Failed to patch ArgoCD Application {app_name}: {e}")
